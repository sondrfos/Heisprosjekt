package udp

import (
	."../definitions"
	"./localip"
	"./bcast"
	"./peers"
	"time"
	"reflect"
)


func UDPInit(UDPoutChan chan Message, UDPinChan chan Message, peerChan chan PeerUpdate) (localIP string) {

	localIP, err := localip.LocalIP()
	if err != nil {
		return ""
	} 
	sendStatus(localIP)
	recieveStatus(peerChan)

	go transmitMessage(UDPoutChan, localIP)
	go recieveMessage(UDPinChan, localIP)

	return localIP
	}	

func MasterInit(peerChan 		chan PeerUpdate, 	isMaster chan bool, 
				peerMasterChan 	chan PeerUpdate, 	localIP string, 
				UDPoutChan 		chan Message, 		masterIDChan chan string,) (masterID string){
	timer := time.NewTimer(time.Millisecond*500).C
	peerInfo := PeerUpdate{}
	For:
	for{
		select{
		case peerInfo = <- peerChan:

		case <- timer:
			break For
		}
	}
	askPeersAboutMaster(peerInfo, localIP, UDPoutChan)
	timer2 := time.NewTimer(time.Second*2).C
	select{
	case masterID = <- masterIDChan:
		break
	case <- timer2:
		masterID = localIP
		isMaster <- true
		masterIDChan <- masterID
		peerMasterChan <- peerInfo
		break
	}
	return masterID
}

func UDPUpkeep(	peerChan 	chan PeerUpdate,	peerMasterChan 	chan PeerUpdate, 
				isMaster 	chan bool, 			masterIDChan 	chan string, 
				UDPoutChan 	chan Message,		masterID 		string, 
				localIP 	string){
	for{
		Upkeep:
			select {
			case peerInfo := <-peerChan:
				companions := peerInfo.Peers
				lostCompanions := peerInfo.Lost
				newCompanion := peerInfo.New
				for _, lostCompanion := range lostCompanions{
					if(lostCompanion == masterID){
						masterID = ""
						for _,companion := range companions{
							if (masterID == ""){
								masterID = companion
							}
							if (companion < masterID){
								masterID = companion
							}
						}
						if( masterID == localIP){
							isMaster <- true
							peerMasterChan <- peerInfo
						}
						masterIDChan <- masterID
					}
				}
				if(masterID == localIP && newCompanion != ""){
					go askAboutMaster(newCompanion, localIP, UDPoutChan)
					ticker := time.NewTicker(time.Second * 1).C 
					i := 0
					select{
					case otherMasterID := <- masterIDChan:
						if(otherMasterID < masterID){
							isMaster <- false
							masterID = otherMasterID
						}
					case <- ticker:
						go askAboutMaster(newCompanion, localIP, UDPoutChan)						//rebroadcasting if there is no reply
						i+=1
						if(i > 5){
							break Upkeep
						}
					}
				}
			}
	}
}

func askPeersAboutMaster(peerInfo PeerUpdate, localIP string, UDPoutChan chan Message){
	for _, companion := range peerInfo.Peers{
		if companion != localIP{
			askAboutMaster(companion, localIP, UDPoutChan)
		}
	}
}

func askAboutMaster(companion string, localIP string, UDPoutChan chan Message) {
	Elevators 	:= make(map[string]Elevator)
	elev := Elevator{}
	elev.Floor = 1
	Elevators[localIP] = elev
	msg 		:= Message{Elevators, localIP, companion, 3}
	UDPoutChan	<- msg
	return
}

func transmitMessage(UDPoutChan chan Message, localIP string){
	transmitChan := make(chan Message)
	echoChan := make(chan Message)
	go bcast.Transmitter(MESSAGEPORT, transmitChan)
	go bcast.Receiver(ECHOPORT, echoChan)
	for{
		select{
		case message := <- UDPoutChan:
			message.SenderID = localIP 										//adding the localIP as senderID
			messageCopy := message 											//Making a copy to avoid channel passing map pointers problems											
			transmitChan <- messageCopy										//transmitting the mssage
			waitForEcho(transmitChan, echoChan, message)					//start new goroutine who waits for echo
		}
	}
}

func recieveMessage(UDPinChan chan Message, localIP string){
	recieveChan := make(chan Message)
	echoChan := make(chan Message)
	go bcast.Receiver(MESSAGEPORT, recieveChan)								//starting a receiver to recieve messages
	go bcast.Transmitter(ECHOPORT, echoChan)								//starting a transmitter to transmit echo
	for{
		select{
		case  message := <- recieveChan:
				if(message.RecieverID == localIP){								//checking to see if the message was ment for you
					echoChan <- message 										//putting out an echo on the echoport
					messageCopy := message										//Making a copy to avoid channel passing map pointers problems
					go func(){
						UDPinChan <- messageCopy 									//transmitting the message back to main and further
					}()
				}		
		}
	}
}

func waitForEcho(transmitChan chan Message, echoChan chan Message, message Message){

	ticker := time.NewTicker(time.Millisecond * 1000).C 					//waiting one second between resends
	i := 0
	for{
		select{								
		case <- ticker:
			transmitChan <- message 										//rebroadcasting if there is no reply										
			i+=1
			if(i > 5){
				return											//if no reply in five seconds assume peer lost and stop the  goroutine
			}
		case echo := <-echoChan:
			if(reflect.DeepEqual(echo.Elevators, message.Elevators) && echo.MsgType == message.MsgType){ 	//checking to see if you recieved the right echo
				return																//when the right echo were recieved, stop the echo
			}
		}
	}
}

func sendStatus(localIP string){
	transmitStatus := make (chan bool)
	go peers.Transmitter(STATUSPORT, localIP, transmitStatus)
}

func recieveStatus(peerChan chan PeerUpdate){
	
	go peers.Receiver(STATUSPORT, peerChan)
}

func CheckInternetConnection(internetConnection chan bool) {
	ticker := time.NewTicker(time.Second).C
	transmitChan := make(chan bool)
	recieveChan := make(chan bool)
	go bcast.Transmitter(CONNCHECKPORT, transmitChan)
	go bcast.Receiver(CONNCHECKPORT, recieveChan)
	for {
		select{
		case <- ticker:
			transmitChan <- true
			timer := time.NewTimer(time.Second).C
			select {
			case <- timer:
				internetConnection <- false
			case <- recieveChan:
				internetConnection <- true
			}
		}
	}
}