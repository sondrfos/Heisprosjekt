package driver
/*
#cgo CFLAGS: -std=gnu11
#cgo LDFLAGS: -lcomedi -lm
#include "io.h"
#include "elev.h"
*/
import "C"


type Elev_button_type_t int

const (
	BUTTON_CALL_UP 		Elev_button_type_t = 0
	BUTTON_CALL_DOWN 	Elev_button_type_t = 1
	BUTTON_COMMAND 		Elev_button_type_t = 2
	
)

type Elev_motor_direction_t int

const (
    DIRN_DOWN 	Elev_motor_direction_t = -1
    DIRN_STOP 	Elev_motor_direction_t = 0
    DIRN_UP 	Elev_motor_direction_t = 1
)


func Init() {
	C.elev_init(0)
}

func Direction(dir Elev_motor_direction_t) {
	C.elev_set_motor_direction(C.elev_motor_direction_t(dir))
}

func Get_floor_sensor() int {
	return int(C.elev_get_floor_sensor_signal())+1
}

func Get_button_signal(button Elev_button_type_t, floor int) int {
	return int(C.elev_get_button_signal(C.elev_button_type_t(button), C.int(floor)))
}

func Get_stop_signal() int {
	return int(C.elev_get_stop_signal())
}

func Get_obstruction() int {
	return int(C.elev_get_obstruction_signal())
}

func Set_floor_indicator(floor int) {
	C.elev_set_floor_indicator(C.int(floor))
}

func Set_button_lamp(button Elev_button_type_t, floor int, value int) {
	C.elev_set_button_lamp(C.elev_button_type_t(button), C.int(floor), C.int(value))
}

func Set_stop_lamp(value int) {
	C.elev_set_stop_lamp(C.int(value))
}

func Set_door_open_lamp(value int) {
	C.elev_set_door_open_lamp(C.int(value))
}