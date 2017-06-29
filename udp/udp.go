package udp

import(
  ."../definitions"
  "./localip"
  "./bcast"
  "./peers"
  "fmt"
  "time"
  "math/rand"
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

  func transmitMessage(UDPoutChan chan Message, localIP string){
  	transmitChan := make(chan Message)
  	echoChan := make(chan Message)
  	go bcast.Transmitter(MESSAGEPORT, transmitChan)
  	go bcast.Receiver(ECHOPORT, echoChan)
  	for{
  		select{
  		case message := <- UDPoutChan:
  			message.SenderID = localIP
  			transmitChan <- message
  			waitForEcho(transmitChan, echoChan, message)
  		}
  	}
  }

  func recieveMessage(UDPinChan chan Message, localIP string){
  	recieveChan := make(chan Message)
  	echoChan := make(chan Message)
  	go bcast.Receiver(MESSAGEPORT, recieveChan)
  	go bcast.Transmitter(ECHOPORT, echoChan)
  	for{
  		select{
  		case  message := <- recieveChan:
  				if(message.RecieverID == localIP){
  					echoChan <- message
  					UDPinChan <- message
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

  func waitForEcho(transmitChan chan Message, echoChan chan Message, message Message){
    fmt.Println("We are waiting for echo")
    tick := time.NewTicker(time.Millisecond*10).C
    it := 0
    for{
      select{
      case <- tick:
        fmt.Println("Echoing")
        transmitChan <- message
        it++
        if(it > 10){
          //throw exception
        }
      case reply := <- echoChan:
        if (testEq(reply.Elevators, message.Elevators) && reply.SenderID == message.SenderID && reply.RecieverID == message.RecieverID && reply.MsgType == message.MsgType){
          fmt.Println("It's a bingo")
          return
        }

      }
    }

  }


  func testEq(a, b []Elevator) bool {

      if a == nil && b == nil {
          return true;
      }

      if a == nil || b == nil {
          return false;
      }

      if len(a) != len(b) {
          return false
      }

      for i := range a {
          if a[i] != b[i] {
              return false
          }
      }

      return true
  }
