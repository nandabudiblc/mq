package main

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	. "github.com/eaciit/mq/client"
	. "github.com/eaciit/mq/msg"
	. "github.com/eaciit/mq/server"
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	var e error
	c, e := NewMqClient("127.0.0.1:7890", time.Second*10)
	handleError(e)
	fmt.Println("Connecting to RPC Server")
	isLoggedIn := c.ClientInfo.IsLoggedIn

	r := bufio.NewReader(os.Stdin)
	ActiveUser := ""
	for !isLoggedIn {
		fmt.Print("UserName: ")
		getUserName, _, _ := r.ReadLine()
		UserName := string(getUserName)
		fmt.Print("Password: ")
		getPassword, _, _ := r.ReadLine()
		Password := string(getPassword)
		msg := MqMsg{Key: UserName, Value: Password}

		Role := ""
		i, e := c.CallToLogin(msg)
		handleError(e)
		if i.Value.(ClientInfo).IsLoggedIn {
			isLoggedIn = true
			Role = i.Value.(ClientInfo).Role
			ActiveUser = i.Value.(ClientInfo).Username
		}

		if isLoggedIn {
			scrMsg := fmt.Sprintf("Login Succesfull, your role is: %s with username: %s ", Role, ActiveUser)

			fmt.Println(scrMsg)
		} else {
			fmt.Println("Login Failed!")
		}
	}

	msg := MqMsg{Key: "INFO", Value: "New Client Connected"}
	c.CallToLog("SetLog", msg)

	status := ""

	for status != "exit" {
		fmt.Print("> ")
		//fmt.Scanln(&command)
		line, _, _ := r.ReadLine()
		command := string(line)
		handleError(e)
		lowerCommand := ""
		if strings.HasPrefix(command, "get") || strings.HasPrefix(command, "set") || strings.HasPrefix(command, "inc") || strings.HasPrefix(command, "gettable") {
			stringsPart := strings.Split(command, "(")
			lowerCommand = strings.ToLower(stringsPart[0])
		} else {
			stringsPart := strings.Split(command, " ")
			lowerCommand = strings.ToLower(stringsPart[0])
		}

		if lowerCommand == "exit" {
			status = "exit"
			c.Close()
		} else if lowerCommand == "kill" {
			c.CallString("Kill", "")
			status = "exit"
			c.Close()
		} else if lowerCommand == "ping" {
			s, e := c.CallString("Ping", "")
			handleError(e)
			fmt.Printf(s)
		} else if lowerCommand == "nodes" {
			results := []Node{}
			e := c.CallDecode("Nodes", "", &results)
			handleError(e)
			fmt.Printf("%v\n", results)
		} else if lowerCommand == "gettable" {
			_, data := parseGetTableCommand(command)
			tableName := data.Key
			ownerName := ActiveUser + "|" + data.Owner
			// if ownerName == "" {
			// 	ownerName = ActiveUser
			// } else {
			// 	ownerName
			// }

			//fmt.Println(tableName)

			msg := MqMsg{Key: tableName, Value: ownerName}
			results := []Table{}
			e := c.CallDecode("GetTable", msg, &results)
			if e != nil {
				fmt.Println("Unable to store message: " + e.Error())
			}

			handleError(e)
			tableContent := fmt.Sprintf("Key\t\t|Value\t\t|Owner\n")
			for i := range results {
				tableContent = tableContent + fmt.Sprintf("%s\t\t|%s\t\t|%s\n", strings.Split(results[i].Key, "|")[2], results[i].Value, results[i].Owner)
			}
			fmt.Println(tableContent)
			//fmt.Printf("%v\n", results)
		} else if lowerCommand == "set" {
			_, data := parseSetCommand(command)
			//owner := c.ClientInfo.Username
			keygenerate := data.BuildKey(data.Owner, data.Table, data.Key)
			value := data.Value
			if data.Value == nil {
				value = " "
			}

			msg := MqMsg{Key: keygenerate, Value: value}
			_, e := c.Call("Set", msg)
			if e != nil {
				fmt.Println("Unable to store message: " + e.Error())
			}

		} else if lowerCommand == "inc" {
			Orikey, incVal := parseIncCommand(command)
			//owner := c.ClientInfo.Username
			m := MqMsg{}
			keygenerate := m.BuildKey("public", "", Orikey)
			//keygenerate := m.BuildKey(owner, "", Orikey)
			msg, e := c.Call("Get", keygenerate)
			if e != nil {
				fmt.Println("No Data with Key : " + keygenerate)
			} else {
				val, _ := strconv.Atoi(incVal)
				valstr := msg.Value.(string)
				newval, _ := strconv.Atoi(valstr)
				xxx := newval + val
				_, e := c.CallInc("Inc", strconv.Itoa(xxx), keygenerate)
				if e != nil {
					fmt.Println("Unable to Increase value, message: " + e.Error())
				}

				//fmt.Printf("Value: %v \n", msg.Value)
			}

		} else if lowerCommand == "get" {
			//--- this to handle get command
			_, data := parseGetCommand(command)
			keyx := strings.Split(data, ",")[0]
			anotherkeys := strings.Split(data, ",")[1:]

			own := ""
			tbl := ""
			for i := 0; i < len(anotherkeys); i++ {
				akey := strings.Split(anotherkeys[i], "=")[0]
				bkey := strings.Split(anotherkeys[i], "=")[1]
				if strings.TrimSpace(akey) == "owner" {
					own = bkey
				}
				if strings.TrimSpace(akey) == "table" {
					tbl = bkey
				}
			}

			own = strings.Trim(own, "\"")
			tbl = strings.Trim(tbl, "\"")

			//m := MqMsg{}
			//keys := m.BuildKey(own, tbl, keyx)

			//commandParts := strings.Split(keyx, " ")
			//key := commandParts[1]

			//fmt.Println("owner:", own)
			//fmt.Println("table:", tbl)

			if tbl != "" {
				keyx = tbl + "|" + keyx
			}
			//fmt.Println("keyx:", keyx)

			valPublic := ""
			valOwner := ""

			//if owner = "", looping 2x, first get as public, second get as specified user
			if own == "" {
				valPublic = getValue("public|"+keyx, c)
				valOwner = getValue(ActiveUser+"|"+keyx, c)
			} else {
				valOwner = getValue(ActiveUser+"|"+keyx, c)
			}

			if valPublic != "" {
				fmt.Println("Value (public) : ", valPublic)
			}
			if valOwner != "" {
				//fmt.Println("Value (owner:"+ActiveUser+"): ", valOwner)
				fmt.Println("Value : ", valOwner)
			}
			if valPublic == "" && valOwner == "" {
				fmt.Println("Unable to store message: Data doesn't exist")
			}

		} else if lowerCommand == "getlog" {
			commandParts := strings.Split(command, " ")
			key := commandParts[1]
			value := strings.Join(commandParts[2:], " ")
			msg := MqMsg{Key: key, Value: value}
			s, e := c.CallString("GetLogData", msg)
			handleError(e)
			fmt.Println(s)
		} else if lowerCommand == "adduser" {
			//--- this to handle set command
			commandParts := strings.Split(command, " ")
			userName := commandParts[1]
			role := commandParts[2]
			password := strings.Join(commandParts[3:], " ")
			msg := MqMsg{Key: userName + "|" + role, Value: password}
			_, e := c.Call("AddUser", msg)
			if e != nil {
				fmt.Println("Unable to store message: " + e.Error())
			}
		} else if lowerCommand == "deleteuser" {
			//--- this to handle set command
			commandParts := strings.Split(command, " ")
			userName := commandParts[1]
			msg := MqMsg{Key: userName, Value: userName}
			i, e := c.Call("DeleteUser", msg)
			if e != nil {
				fmt.Println("Unable to store message: " + e.Error())
			} else {
				fmt.Println(i.Value.(string))
			}
		} else if lowerCommand == "changepassword" {
			//--- this to handle set command
			commandParts := strings.Split(command, " ")
			userName := commandParts[1]
			password := strings.Join(commandParts[2:], " ")
			msg := MqMsg{Key: userName, Value: password}
			i, e := c.Call("ChangePassword", msg)
			if e != nil {
				fmt.Println("Unable to store message: " + e.Error())
			} else {
				fmt.Println(i.Value.(string))
			}
		} else if lowerCommand == "getlistusers" {
			s, e := c.CallString("GetListUsers", "")
			handleError(e)
			fmt.Printf(s)
		} else {
			errorMsg := "Unable to find command " + command
			//c.CallToLog(errorMsg,"ERROR")
			msg := MqMsg{Key: "ERROR", Value: errorMsg}
			_, e := c.CallToLog("SetLog", msg)
			if e != nil {
				fmt.Println("Unable to store message: " + e.Error())
			}

			fmt.Println(errorMsg)
		}
	}

}

func handleError(e error) {
	if e != nil {
		fmt.Println(e.Error())
		os.Exit(100)
	}
}

func commandToObject(data string) MqMsg {
	m := MqMsg{}

	m.Key = strings.TrimSpace(strings.Split(data, ",")[0])
	anotherkeys := strings.Split(data, ",")[1:]

	for i := 0; i < len(anotherkeys); i++ {
		obj := strings.ToLower(strings.TrimSpace(anotherkeys[i]))

		if i == 0 && !strings.Contains(obj, "=") {
			m.Value = anotherkeys[i]
		}

		if strings.Split(obj, "=")[0] == "table" {
			m.Table = strings.Trim(strings.Split(obj, "=")[1], "\"")

		}
		if strings.Split(obj, "=")[0] == "owner" {
			m.Owner = strings.Trim(strings.Split(obj, "=")[1], "\"")

		}
		if strings.Split(obj, "=")[0] == "duration" {
			i64, _ := strconv.ParseInt(strings.Split(obj, "=")[1], 10, 0)
			m.Duration = i64

		}
		if strings.Split(obj, "=")[0] == "permission" {
			m.Permission = strings.Trim(strings.Split(obj, "=")[1], "\"")

		}
		if strings.Split(obj, "=")[0] == "value" {
			m.Value = strings.Split(obj, "=")[1]

		}

	}
	return m
}

func parseSetCommand(command string) (string, MqMsg) {
	match, _ := regexp.MatchString("set()", command)
	if match == true {
		splitSet := strings.Split(command, "set(")[1]
		data := strings.TrimRight(splitSet, ")")

		m := commandToObject(data)

		return "set", m

	} else {
		return "set", MqMsg{}
	}
}

func parseIncCommand(command string) (string, string) {
	match, _ := regexp.MatchString("inc()", command)
	if match == true {
		splitSet := strings.Split(command, "inc(")[1]
		data := strings.TrimRight(splitSet, ")")
		return strings.Split(data, ",")[0], strings.Split(data, ",")[1]

	} else {
		return "", ""
	}
}
func parseGetTableCommand(command string) (string, MqMsg) {
	match, _ := regexp.MatchString("gettable()", strings.ToLower(command))
	if match == true {
		splitSet := strings.Split(command, "gettable(")[1]
		data := strings.TrimRight(splitSet, ")")

		m := commandToObject(data)

		return "gettable", m

	} else {
		return "gettable", MqMsg{}
	}
}

func parseGetCommand(command string) (string, string) {
	match, _ := regexp.MatchString("get()", command)
	if match == true {
		splitSet := strings.Split(command, "get(")[1]
		data := strings.TrimRight(splitSet, ")")
		return "get", data

	} else {
		return "get", ""
	}
}

func getValue(key string, c *MqClient) string {
	//fmt.Println("key:", key)
	msg, e := c.Call("Get", key)
	if e != nil {
		//fmt.Println("Unable to store message: " + e.Error())
		return ""
	} else {
		return msg.Value.(string)
		//fmt.Printf("Value: %v \n", msg.Value)
	}
}
