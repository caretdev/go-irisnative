package main

import (
	"fmt"
	"os"

	"github.com/caretdev/go-irisnative/src/connection"
)

func main() {
	var addr = "localhost:3082"
	var namespace = "%SYS"
	var login = "_SYSTEM"
	var password = "SYS"

	connection, err := connection.Connect(addr, namespace, login, password)
	if err != nil {
		println("Connection failed:", err.Error())
		os.Exit(1)
	}
	defer connection.Disconnect()

	fmt.Println("Connection established")
	serverVersion, _ := connection.ServerVersion()
	fmt.Println(serverVersion)

	// var request string
	// connection.ClasMethod("%CSP.Request", "%New", &request)
	// fmt.Println(request)
	// var result string
	// connection.ClasMethodVoid("%Studio.General", "Execute", "zw ##class(%CSP.Request).%New()")
	// fmt.Println(result)
	// var alerts string
	// connection.ClasMethod("SYS.Monitor.SAM.Sensors", "Alerts", &alerts)
	// fmt.Println(alerts)

	// var metrics string
	// connection.ClasMethod("SYS.Monitor.SAM.Sensors", "PrometheusMetrics", &metrics)
	// re := regexp.MustCompile(`^(\w+)(?:{(?:id="([^"]+)")?[^}]*})? (\d+|\d*\.\d+)$`)
	// for _, l := range strings.Split(metrics, "\n") {
	//   fmt.Printf("%#v\n", re.FindStringSubmatch(l)[1:])
	// }

	// // Kill ^A
	// connection.GlobalKill("A")
	// // Set ^A(1) = 1
	// connection.GlobalSet("A", 1, 1)
	// // Set ^A(1, 2) = "test"
	// connection.GlobalSet("A", "test", 1, 1)
	// // Set ^A(1, "2", 3) = "123"
	// connection.GlobalSet("A", 123, 1, "a", 3)
	// // Set ^A(2, 1) = "21test"
	// connection.GlobalSet("A", "21test", 2, 1)
	// // Set ^A(3, 1) = "test31"
	// connection.GlobalSet("A", "test31", 3, 1)

	// var globalFull = func(global string, subs ...interface{}) string {
	// 	return fmt.Sprintf("^A(%v)", strings.Trim(strings.Join(strings.Split(fmt.Sprintf("%+v", subs), " "), ", "), "[]"))
	// }
	// var queryGlobal func(global string, subs ...interface{})
	// queryGlobal= func(global string, subs ...interface{}) {
	// 	for i := ""; ; {
	// 		if hasNext, _ := connection.GlobalNext("A", &i, subs...); !hasNext {
	// 			break
	// 		}
	// 		var allSubs = []interface{}{i}
	// 		allSubs = append(subs, allSubs...)
	// 		hasValue, hasSubNode := connection.GlobalIsDefined("A", allSubs...)
	// 		if hasValue {
	// 			var value string
	// 			connection.GlobalGet("A", &value, allSubs...)
	// 			fmt.Printf("%v = %#v\n", globalFull("A", allSubs...), value)
	// 		}
	// 		if hasSubNode {
	// 			queryGlobal("A", allSubs...)
	// 		}
	// 	}
	// }

	// queryGlobal("A")

}
