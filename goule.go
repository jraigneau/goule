package main

import (
    "log"
     "fmt"
    "gopkg.in/telegram-bot-api.v4"
    "github.com/influxdata/influxdb/client/v2"
     "encoding/json"
)


const (
    username = "grafana"
    password = "paint"
    addr = "http://obelix:8086"
)


//Récupère les dernières valeurs de température
func getTemperatures() string {
	
	q := fmt.Sprintf("SELECT * FROM temperature ORDER BY time DESC LIMIT 15")
	res, err := queryDB(q,"tempDB")
	if err != nil {
    	log.Fatal("Error: ",err)
	}

	var temperatures = make(map[string]string)
	for row := range res[0].Series[0].Values {
	    name := res[0].Series[0].Values[row][1].(string)
	    val := res[0].Series[0].Values[row][2].(json.Number).String()
	    _,ok := temperatures[name] //vérifie la présence de la pièce dans la map
	    if !ok {
	    	temperatures[name] = val
	    }
	}
	var result = "``` Les températures des pièces sont:"
	for room := range temperatures {
		result = fmt.Sprintf("%s \n %s: %s°C",result,room,temperatures[room])
	}
	result = fmt.Sprintf("%s ```",result)

	return result
}

func getConsoElectrique() string {

	q := fmt.Sprintf("SELECT * FROM energy ORDER BY time DESC LIMIT 1")
	res, err := queryDB(q,"electricity")
	if err != nil {
    	log.Fatal("Error: ",err)
	}

	day_energy := res[0].Series[0].Values[0][1].(json.Number).String()
	instant_energy := res[0].Series[0].Values[0][2].(json.Number).String()
	
	return fmt.Sprintf("``` Actuellement la consommation instantanée est de %sW et le cumul est de %skW ```",instant_energy,day_energy)

}


//Analyse le message
func msgAnalysis(input string) string {
    output := "Désolé, je n'ai pas reconnu la commande"
    switch input {
    	case "/start": output = "``` Bonjour Maître, que puis-je pour vous aujourd'hui?```"
    	case "/help": output = "``` Je m'appelle Goule, et je sers la maison de mon Maître```"
    	case "/temp": output = getTemperatures()
    	case "/conso": output = getConsoElectrique()
    	default: output = "``` Désolé, je n'ai pas reconnu la commande```"
    }
    return output
}


// queryDB convenience function to query the database
func queryDB(cmd string, MyDB string) (res []client.Result, err error) {
    
	log.Printf("Connection à influxDB")
	clnt, err := client.NewHTTPClient(client.HTTPConfig{
        Addr: addr,
        Username: username,
        Password: password,
    })
    if err != nil {
        log.Fatalln("Error: ", err)
    }


    q := client.Query{
        Command:  cmd,
        Database: MyDB,
    }
    response, err := clnt.Query(q)
    if err != nil {
    	log.Fatalln("Error: ", err)
    }
    if response.Error() != nil {
        log.Fatalln("Error: ", response.Error())
    }
    res = response.Results
    log.Println(response.Results)
    return res, nil
}


func main() {



    bot, err := tgbotapi.NewBotAPI("266659220:AAGB3cokOQu6ZswK9xt6EIhnPy7Gs1CpoWs")
    if err != nil {
        log.Panic(err)
    }
    master := "sirjuh"

    bot.Debug = true

    log.Printf("Authorized on account %s", bot.Self.UserName)

    u := tgbotapi.NewUpdate(0)
    u.Timeout = 60

    updates, err := bot.GetUpdatesChan(u)

    for update := range updates {
        if update.Message == nil {
            continue
        }
        if update.Message.From.UserName == master { //vérifie username
            log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)
            input := update.Message.Text
            output := msgAnalysis(input)
        	msg := tgbotapi.NewMessage(update.Message.Chat.ID, output)
        	msg.ParseMode = "Markdown" 
        	bot.Send(msg)
        } else {
        	log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)
        	msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Désolé je n'obéis qu'à mon Maître")
        	bot.Send(msg)
        }

    }
}