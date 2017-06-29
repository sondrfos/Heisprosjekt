package main

import (
	. "./definitions"
	"./localLift"
	"./udp"
	"./master"
	."./watchDog"
	"fmt"
)




func main(){
	WatchDog()

	go stateMachine()
	go Heartbeat()

	StartBackup()	

	select{}
}



func stateMachine(){
	//Make channels
	UDPoutChan 		:= make(chan Message)
	UDPinChan 		:= make(chan Message)
	masterMessage 	:= make(chan Message)
	peerChan 		:= make(chan PeerUpdate)
	peerMasterChan 	:= make(chan PeerUpdate)
	elevOut 		:= make(chan Elevator)
	elevIn 			:= make(chan Elevator)
	currentElevState:= make(chan Elevator)
	internetConnect := make(chan bool)
	isMaster 		:= make(chan bool)
	masterIDChan 	:= make(chan string)
	stateChan 		:= make(chan string)
	masterID 		:= ""
	localIP			:= ""
	state 			:= "Initialize elev"
	currentState 	:= Elevator{}
	initialized 	:= false

	for{
		StateMachine:
		switch state {	
		case "Initialize elev":

			go udp.CheckInternetConnection(internetConnect)
			go localLift.Elev_driver(elevIn, elevOut)

			state = "Initialize"

		case "Initialize":	

			localIP = udp.UDPInit(UDPoutChan, UDPinChan, peerChan)
			if( localIP == ""){ 
				state = "No internet"
				break 
			}
			if (!initialized){
				go master.MasterLoop(isMaster, masterMessage, peerMasterChan, UDPoutChan)
				go treatMessages(UDPinChan, UDPoutChan, masterMessage, masterIDChan, elevIn, elevOut, currentElevState, stateChan, localIP)
				masterID = udp.MasterInit(peerChan, isMaster, peerMasterChan, localIP, UDPoutChan, masterIDChan)
				
				
				go udp.UDPUpkeep(peerChan, peerMasterChan, isMaster, masterIDChan, UDPoutChan, masterID, localIP)

				initialized = true
			}

			state = "Normal operation"
			stateChan <- state

		case "Normal operation":

			currentStateCopy := currentState 							//Making a copy to avoid channel passing map pointers problems
			elevOut <- currentStateCopy

			for{
				select{
				case internet := <- internetConnect:
					if(!internet){
						state = "No internet"
						stateChan <- state
						break StateMachine
					}
				case currentState = <- currentElevState:
				}
			}

		case "No internet":

			internetConnection 	:= make(chan bool)
			currentStateChan 	:= make(chan Elevator)
			currentState.Order 	 = currentState.Light 

			go localLift.LocalMode(internetConnection, currentStateChan, elevIn, elevOut, currentState)
			for{
				select{
				case internet := <- internetConnect:
					if(internet){
						state = "Initialize"
						internetConnection <- true
						select{
						case currentState = <- currentStateChan:
							fmt.Println(currentState)
							currentState.Order = currentState.Light 
							break StateMachine
						}
					}
				}		
			}
		}
	}
}


func treatMessages(	UDPinChan 			chan Message, 	UDPoutChan 		chan Message, 
					masterMessage 		chan Message, 	masterIDChan 	chan string, 
					elevIn 				chan Elevator, 	elevOut 		chan Elevator, 
					currentElevState 	chan Elevator,	stateChan 		chan string,
					localIP 		string){

	Elevators 				:= make(map[string]Elevator)
	messageBackup 			:= Message{Elevators, "", "", 0}
	masterID 				:= ""
	state 					:= ""
	messageBackup.Elevators[localIP] = Elevator{}
	for{
		if state == "No internet"{
			select{
			case state = <- stateChan:
			}
		}
		select{
		case tempMessage := <- UDPinChan:
			if(checkMessageValidity(tempMessage.Elevators[tempMessage.SenderID])){
				messageBackup = tempMessage
				if (messageBackup.MsgType == 1 && localIP == masterID){
					messageCopy := messageBackup    								//Making a copy to avoid channel passing map pointers problems
					masterMessage <- messageCopy
				} else if (messageBackup.MsgType == 2){
					elevIn <- messageBackup.Elevators[localIP]
					currentElevState <- messageBackup.Elevators[localIP]
				} else if (messageBackup.MsgType == 3){
					master.AmIMaster(messageBackup, masterID, UDPoutChan, localIP)
				} else if (messageBackup.MsgType == 4){
					masterIDChan <- messageBackup.SenderID
					masterID = messageBackup.SenderID
				}
			}
		case masterID = <- masterIDChan:

		case elev_status := <- elevOut:
			messageBackup.Elevators[localIP] = elev_status
			messageBackup.MsgType = 1
			messageBackup.RecieverID = masterID
			messageCopy := messageBackup    									//Making a copy to avoid channel passing map pointers problems
			UDPoutChan <- messageCopy

		case state = <- stateChan:
		}
	}
}


func checkMessageValidity(message Elevator) bool{

	if(message.Floor < 1) {return false} 
	if(message.Floor > 4) {return false}
	for _, element := range message.Order.IntButtons{
		if element > 1 || element < 0 {
			return false
		}
	}
	for _, element := range message.Order.ExtUpButtons{
		if element > 1 || element < 0 {
			return false
		}
	}
	for _, element := range message.Order.ExtDwnButtons{
		if element > 1 || element < 0 {
			return false
		}
	}
	return true
}