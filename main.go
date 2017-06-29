package main

import(
  ."./definitions"
  "./udp"
  "time"
  "fmt"
)

func main(){
  UDPoutChan := make(chan Message)
  UDPinChan := make(chan Message)
  peerChan := make(chan PeerUpdate)
  localIP := udp.UDPInit(UDPoutChan, UDPinChan, peerChan)
  var msg Message
  msg.RecieverID = localIP
  for{
    tick := time.NewTicker(time.Millisecond*1000).C
    select{
    case <- tick:
      UDPoutChan <- msg
    case reply := <- UDPinChan:
      fmt.Println(reply)
    }
  }
}
