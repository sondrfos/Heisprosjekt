package localLift


import (
	."../definitions"
	. "../driver"
	"../master"
	"time"
)




func LocalMode(	internetConnection chan bool, currentStateChan chan Elevator, 
				elevIn chan Elevator, elevOut chan Elevator, currentState Elevator) {
	elevators 		:= make(map[string]Elevator)
	localIP  		:= ""
	change1 		:= false
	change2			:= false

	elevators[localIP] = currentState
	elevIn <- currentState
	for{
		select{
		case elevator := <- elevOut:
			elevators[localIP] = elevator
			elevators, change1 = master.IsTheElevatorFinished(elevators, localIP)
			elevators, change2 = master.CalculateOptimalElevator(elevators, localIP)
			currentState = elevators[localIP]
			if(change1 || change2){
				elevIn <- currentState
				change1, change2 = false, false
			}
		case <- internetConnection:
			currentStateChan <- elevators[localIP] 
			return
		}
	}
}

func Elev_driver(incm_elev_update chan Elevator, out_elev_update chan Elevator) int {

	//---Create channels------------------------------
	target 		:= make(chan int)
	lights 		:= make(chan Buttons)
	statusIn 	:= make(chan Elevator)
	statusOut 	:= make(chan Elevator)


	//---Init of driver-------------------------------
	init_result := elev_init(target,lights,statusIn,statusOut)
	if init_result == 0 {
		return 0 //The elevator failed to initialize
	}

	time.Sleep(time.Second)
	go sendElevOut(out_elev_update, statusOut)
	//---Normal operation-----------------------------
	for {
		select {
		case local_lift := <-incm_elev_update:
			lights <- local_lift.Light
			statusIn <- local_lift 
			target <- local_lift.Queue[0]	
		}
	}
}

func sendElevOut (out_elev_update chan Elevator, statusOut chan Elevator){
	for{
		select{
		case lift_status := <-statusOut:
			out_elev_update <- lift_status   
		}			
	}
}

func elev_init(target chan int, lights chan Buttons, statusIn chan Elevator, statusOut chan Elevator) int { //Initilizes the elevator and the IO. Returns 0 if init fails. Returns 1 otherwise.
	Init()
	elev_clear_lights()
	floor_sense 	:= make(chan int)
	directionChan 	:= make(chan Elev_motor_direction_t)

	go elev_status_checker(statusIn, statusOut, directionChan)
	elev_go(DIRN_DOWN)
	go elev_poll_floor_sensor(floor_sense)
	for {
		select{
		case position :=<- floor_sense:
			if position != 0{
				elev_go(DIRN_STOP)
				//---Start light controller and status checker----
				go elev_light_controller(lights)
				go elev_go_to_floor(target, directionChan)
				return 1
			}
		}
	}	
}

func elev_go(dir Elev_motor_direction_t) { //Sets the apporpriate direction for the elevator and sends it on its way.
	Direction(dir)
}

func elev_calculate_dir(target int,floor int) (Elev_motor_direction_t){
	if target < floor { return DIRN_DOWN }
	if target > floor { return DIRN_UP }  
	return DIRN_STOP
}

func elev_go_to_floor(target chan int, directionChan chan Elev_motor_direction_t){
	done_stopping 	:= make(chan bool)
	floor 			:= make(chan int)
	stopping 		:= false
	last_dir 		:= DIRN_STOP
	ticker 			:= time.NewTicker(time.Second).C


	go elev_poll_floor_sensor(floor)
	current_floor:= <- floor
	current_target := 0

	for{
		select{
		case current_target = <- target:
		case position:= <- floor:
			if(position == 0){ 
				continue 
			} else{
				current_floor = position
			}
		case <- done_stopping:
			stopping = false
		case <- ticker:
			if (!stopping && (current_target <= FLOORS && current_target > 0)){
				dir := elev_calculate_dir(current_target,current_floor)
				elev_go(dir)
				if(dir != last_dir){
					last_dir = dir
					directionChan <- dir
				}
			}
		}
		if (!stopping && (current_target <= FLOORS && current_target > 0)){
			dir := elev_calculate_dir(current_target,current_floor)
			elev_go(dir)
			if(dir != last_dir){
				last_dir = dir
				directionChan <- dir
			}
		}
		if(current_target == current_floor) && (current_target <= FLOORS && current_target > 0){
			stopping = true
			elev_go(DIRN_STOP)
			directionChan <- DIRN_STOP
			go elev_stop_at_floor(done_stopping)
		}
	}
}

func elev_stop_at_floor(done chan bool) { 
	Set_door_open_lamp(1)
	time.Sleep(time.Second * 3)
	Set_door_open_lamp(0)
	done <- true 
	return
}

func elev_set_floor_light(floor int) {
	Set_floor_indicator(floor-1)
}

func elev_poll_floor_sensor(floor_sense chan int){ //Returns the floor if the elevator is there, otherwise returns 0
	floor := -1
	last_floor := -1

	for{
		time.Sleep(time.Millisecond*50)
		floor = Get_floor_sensor()
		if(last_floor != floor){
			last_floor = floor
			floor_sense <- floor
		}
	}
}

func elev_check_buttons(button_presses chan Buttons) {
	button_inputs := Buttons{}
	dummy_inputs  := Buttons{}

	for {
		//Reads the internal orders
		for i := range dummy_inputs.IntButtons{
			dummy_inputs.IntButtons[i] = Get_button_signal(BUTTON_COMMAND, i)
		}

		//Reads external up orders
		for i:= range dummy_inputs.ExtUpButtons{
			dummy_inputs.ExtUpButtons[i] = Get_button_signal(BUTTON_CALL_UP, i)
		}

		//Reads external down orders
		for i:= range dummy_inputs.ExtDwnButtons{
			dummy_inputs.ExtDwnButtons[i] = Get_button_signal(BUTTON_CALL_DOWN, i+1)
		}

		if button_inputs != dummy_inputs {
			button_inputs = dummy_inputs
			button_presses <- button_inputs	
		}
		time.Sleep(time.Millisecond*50)
	}
}

func elev_light_controller(orders chan Buttons) {
	floor_light := make(chan int)
	go elev_poll_floor_sensor(floor_light)
	for {
		select {
		case lights := <-orders:
			for i, value := range lights.IntButtons{
				Set_button_lamp(BUTTON_COMMAND, i, value)
			}
	
			for i, value := range lights.ExtUpButtons{
				Set_button_lamp(BUTTON_CALL_UP, i, value)
			}

			for i, value := range lights.ExtDwnButtons{
				Set_button_lamp(BUTTON_CALL_DOWN, i+1, value)
			}
		case floor := <- floor_light:
			if floor != 0{
				elev_set_floor_light(floor)
			}
		}
	}
}

func elev_clear_lights(){
	for i :=0; i < 4; i++{
		Set_button_lamp(BUTTON_COMMAND, i, 0)
		if(i<3){ Set_button_lamp(BUTTON_CALL_UP, i, 0) }
		if(i>0){Set_button_lamp(BUTTON_CALL_DOWN, i, 0)}
	}
	Set_door_open_lamp(0)
	Set_stop_lamp(0)
}

func elev_status_checker(statusIn chan Elevator, statusOut chan Elevator, direction chan Elev_motor_direction_t) {
	status_elev 	:= Elevator{}
	ticker 			:= time.NewTicker(time.Second).C
	buttons 		:= make(chan Buttons)
	floor_sense 	:= make(chan int)

	go elev_poll_floor_sensor(floor_sense)
	go elev_check_buttons(buttons)

	for {
		select {
		case presses 	:= <-buttons:
			status_elev.Order = presses
			statusOut <- status_elev
			status_elev.Order = Buttons{}
		case dir 		:= <- direction:
			status_elev.Direction = int(dir)
			statusOut <- status_elev
		case 			   <-ticker:
			statusOut <- status_elev
		case position  	:= <- floor_sense:
			if position != 0 { status_elev.Floor = position }
			status_elev.Position = position
			statusOut <- status_elev
		case status_elev = <- statusIn:
		}
	}
}