package watchDog

import(
	"io/ioutil"
	"time"
	"os"
	"os/exec"
	."../definitions"
	"fmt"
)


func WatchDog(){
	for{
		info,err:= os.Stat(FILENAME)
		if err != nil{
			fmt.Println(err.Error())
		}

		if time.Since(info.ModTime()) > (time.Millisecond*500){
			return
		}
		time.Sleep(time.Millisecond*100)
	}
}

func Heartbeat(){
	tick := time.NewTicker(time.Millisecond*100).C
	for{
		select{
		case <-tick:
			s := "Hey,thats pretty good"
			b := []byte(s)
			ioutil.WriteFile(FILENAME, b, 0644)
		}
	}
}

func StartBackup(){
	cmd := exec.Command("gnome-terminal", "-x","go", "run", "main.go")
	cmd.Run()
}