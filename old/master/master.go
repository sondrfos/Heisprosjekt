package master

import (
	."../definitions"
)



func MasterLoop(isMaster 	chan bool, 			masterMessage 	chan Message, 
				peerChan 	chan PeerUpdate, 	UDPoutChan 		chan Message){
	

	slaves := PeerUpdate{}
	messageBackup := Message{}
	for{
		select{
		case iMaster := <- isMaster:
			if(iMaster){
				Master:
				for{
					select{									
					case iMaster = <- isMaster:
						if(!iMaster){
							break Master
						}
					case messageBackup = <- masterMessage:
						senderID := messageBackup.SenderID
						change1 := false
						change2 := false
						if (messageBackup.MsgType == 1){
							messageBackup.Elevators, change1 = IsTheElevatorFinished(messageBackup.Elevators, senderID)
							messageBackup.Elevators, change2 = CalculateOptimalElevator(messageBackup.Elevators, senderID)
							if (change1 || change2) {
								for _,slave := range slaves.Peers{
									messageBackup.RecieverID = slave
									messageBackup.MsgType = 2
									messageCopy := messageBackup		//Making a copy to avoid channel passing map pointers problems
									UDPoutChan <- messageCopy
								}
							}
						}
					case slaves = <- peerChan:
						messageBackup.MsgType = 2
						messageBackup.RecieverID = slaves.New
						messageCopy := messageBackup			//Making a copy to avoid channel passing map pointers problems
						UDPoutChan <- messageCopy
					}
				}
			}
		}
	}
}

func IsTheElevatorFinished(slaves map[string]Elevator, senderIP string) (map[string]Elevator, bool){
	slavePointer := make(map[string]*Elevator)
	var slavetemp [ELEVATORS]Elevator
	change := false
	i :=0

	//making a map with pointers that is possible to change wtf?
	for key := range slaves{
		slavetemp[i] = slaves[key]
		slavePointer[key] = &(slavetemp[i])
		i++
	}
	
	sender := *(slavePointer[senderIP])
	position := sender.Position
	if (position != 0){
		if(position == sender.Queue[0] && sender.Direction == 0){
			change = true 
			for i := range sender.Queue{
				if (i == (len(sender.Queue)-1)){
					sender.Queue[i] = 0
 				} else{
					sender.Queue[i] = sender.Queue[i+1] 
				}
			}
			if (position == 1){
				sender.Light.ExtUpButtons[(position-1)] = 0
			} else if (position == 4){
				sender.Light.ExtDwnButtons[(position-2)] = 0
			} else{ 
				sender.Light.ExtUpButtons[(position-1)] = 0
				sender.Light.ExtDwnButtons[(position-2)] = 0
			}
			sender.Light.IntButtons[position-1] = 0
		}

	}
	slavePointer[senderIP] = &sender

	//converting the map back to normal map without pointers
	elementMap := make(map[string]Elevator)
	for key := range slavePointer{
		elementMap[key] = *slavePointer[key]
	}
	return elementMap, change
}

func CalculateOptimalElevator(slaves map[string]Elevator, senderIP string) (map[string]Elevator, bool){
	//Println("Calculating optimal elevator")
	leastCostID := ""
	firstZero := 0
	slavePointer := make(map[string]*Elevator)
	change := false
	var slavetemp [ELEVATORS]Elevator
	i :=0

	//making a map with pointers that is possible to change
	for key,element := range slaves{
		slavetemp[i] = element
		slavePointer[key] = &(slavetemp[i])
		i++
	}
	senderElevator := (slavePointer[senderIP])
	orders := senderElevator.Order

	//Calculate optimal elevator for external up orders
	for i,order := range orders.ExtUpButtons{
		if(order == 0){
			continue
		} else{
			change = true
			leastCostID, firstZero = calculateOptimalElevatorAssignment(slavePointer, i+1)
			optimalSlave := slavePointer[leastCostID]
			(*optimalSlave).Order.ExtUpButtons[i] = 0
			for _, element := range slavePointer{
				(*element).Light.ExtUpButtons[i] = 1
			}
			if(firstZero == -1){
				continue
			} else{
				(*optimalSlave).Queue[firstZero] = i+1
			}
		}
	}
	//Calculate optimal elevator for external down orders
	for i,order := range orders.ExtDwnButtons{
		if(order == 0){
			continue
		} else{
			change = true
			leastCostID, firstZero = calculateOptimalElevatorAssignment(slavePointer, i+2)
			optimalSlave := slavePointer[leastCostID]
			(*optimalSlave).Order.ExtDwnButtons[i] = 0
			for _, element := range slavePointer{
				(*element).Light.ExtDwnButtons[i] = 1
			}
			if(firstZero == -1){
				continue
			} else{
				(*optimalSlave).Queue[firstZero] = i+2
			}
		}
	}
	//Give internal orders to the right elevator
	senderElevator = (slavePointer[senderIP])	
	for i,order := range orders.IntButtons{
		if (order == 0){
			continue
		} else {
			change = true
			senderElevator.Light.IntButtons[i] = 1
			senderElevator.Order.IntButtons[i] = 0
			for j,queueElement := range senderElevator.Queue{
				if( i+1 == queueElement){
					break
				} 
				if (queueElement == 0){
					senderElevator.Queue[j] = i+1
					break
				}
			}
		}
	}

	elementMap := make(map[string]Elevator)
	for key, element := range slavePointer{
		elementMap[key] = *element
	}
	return elementMap, change
}

func calculateOptimalElevatorAssignment(slaves map[string]*Elevator, order int) (string, int){
	leastCostID := ""
	leastCost := -1
	cost := 0
	firstZero := -2
	leastCostFirstZero := 0
	lastElement := 0
	for ip,slave := range slaves{
		for i,queueElement := range slave.Queue{
			if( order == queueElement){
				return ip, -1
			}
			if(queueElement == 0){
				if(firstZero == -2){
					firstZero = i
				}
				continue
			} else{
				if(i == 0){ lastElement = slave.Floor}else
				{lastElement = slave.Queue[i-1]}
				cost = cost + abs(queueElement - lastElement)
			}
		}
		cost = cost + abs(slave.Floor-order)
		if(leastCost == -1 || leastCost > cost){
			leastCost = cost
			leastCostID = ip
			leastCostFirstZero = firstZero
		}
		firstZero = -2
		cost = 0
	}
	return leastCostID, leastCostFirstZero
}

func AmIMaster(message Message, masterID string, UDPoutChan chan Message, localIP string){
	if(masterID == localIP){
		elev := Elevator{}
		elev.Floor = 1
		message.MsgType = 4
		message.Elevators[localIP] = elev
		message.RecieverID = message.SenderID
		message.SenderID = localIP
		messageCopy := message 							//Making a copy to avoid channel passing map pointers problems
		UDPoutChan <- messageCopy
	}
}

func abs(x int) int {
    if x < 0 {
        return -x
    }
    return x
}