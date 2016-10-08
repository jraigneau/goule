package main

import (
    "log"
     "fmt"
    "gopkg.in/telegram-bot-api.v4"
    "github.com/influxdata/influxdb/client/v2"
     "encoding/json"
     "math"
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
	var result = "Les températures des pièces sont:"
	for room := range temperatures {
		result = fmt.Sprintf("%s\n- %s: *%s°C*",result,room,temperatures[room])
	}


	q1 := fmt.Sprintf("select value from hygrometrie order by time desc limit 1")
	res1, err := queryDB(q1,"hygroDB")
	if err != nil {
    	log.Fatal("Error: ",err)
	}

	hygroSDB := res1[0].Series[0].Values[0][1].(json.Number).String()

	result = fmt.Sprintf("%s\net le degré d'humidité dans le SdB est de *%v%%*.",result,hygroSDB)

	return result
}

//Renvoie la consommation électrique
func getConsoElectrique() string {

	q := fmt.Sprintf("SELECT * FROM energy ORDER BY time DESC LIMIT 1")
	res, err := queryDB(q,"electricity")
	if err != nil {
    	log.Fatal("Error: ",err)
	}

	day_energy := res[0].Series[0].Values[0][1].(json.Number).String()
	instant_energy := res[0].Series[0].Values[0][2].(json.Number).String()
	
	return fmt.Sprintf("Actuellement la consommation instantanée est de *%sW* et le cumul est de *%skW*.",instant_energy,day_energy)

}

//Renvoie le traffic routier
func getTraffic() string {

	q := fmt.Sprintf("SELECT trafficDelayInSeconds,travelTimeInSeconds FROM traffic where \"name\"='julien' ORDER BY time DESC LIMIT 1")
	res, err := queryDB(q,"trafficy")
	if err != nil {
    	log.Fatal("Error: ",err)
	}

	trafficDelayInSecondsJR := res[0].Series[0].Values[0][1].(json.Number).String()
	travelTimeInSecondsJR := res[0].Series[0].Values[0][2].(json.Number).String()

	q1 := fmt.Sprintf("SELECT trafficDelayInSeconds,travelTimeInSeconds FROM traffic where \"name\"='laurence' ORDER BY time DESC LIMIT 1")
	res1, err := queryDB(q1,"trafficy")
	if err != nil {
    	log.Fatal("Error: ",err)
	}

	trafficDelayInSecondsLR := res1[0].Series[0].Values[0][1].(json.Number).String()
	travelTimeInSecondsLR := res1[0].Series[0].Values[0][2].(json.Number).String()
	
	return fmt.Sprintf("Actuellement il faut *%vmin* pour aller au PMU (*%vmin* de bouchon) et *%vmin* pour aller chez Aviva (*%vmin* de bouchon).",travelTimeInSecondsJR,trafficDelayInSecondsJR,travelTimeInSecondsLR,trafficDelayInSecondsLR)

}

//Renvoie les métriques autour du trafic internet @home
func getInternet() string {

	q := fmt.Sprintf("SELECT mean(\"rx\")/1000 FROM traffic where \"interface\" = 'pppoe-wan6' and time > now() - 5m")
	res, err := queryDB(q,"traffic")
	if err != nil {
    	log.Fatal("Error: ",err)
	}
	mean_rx, err := res[0].Series[0].Values[0][1].(json.Number).Float64()
	if err != nil {
    	log.Fatal("Error: ",err)
	}
	mean_rx = math.Floor(mean_rx)

	q2 := fmt.Sprintf("SELECT mean(\"tx\")/1000 FROM traffic where \"interface\" = 'pppoe-wan6' and time > now() - 5m")
	res2, err := queryDB(q2,"traffic")
	if err != nil {
    	log.Fatal("Error: ",err)
	}
	mean_tx, err := res2[0].Series[0].Values[0][1].(json.Number).Float64()
	if err != nil {
    	log.Fatal("Error: ",err)
	}
	mean_tx = math.Floor(mean_tx)

	q3 := fmt.Sprintf("SELECT mean(\"value\") FROM ping where \"site\" = 'google' and time > now() - 5m")
	res3, err := queryDB(q3,"uptime")
	if err != nil {
    	log.Fatal("Error: ",err)
	}
	uptime, err := res3[0].Series[0].Values[0][1].(json.Number).Float64()
	if err != nil {
    	log.Fatal("Error: ",err)
	}
	uptime = math.Floor(uptime)

	var result = fmt.Sprintf("Sur les 5 dernières minutes, Le trafic entrant moyen est de *%vKb/s* et de *%vKb/s* en sortie. La moyenne du ping vers google est *%vms*.",mean_rx,mean_tx,uptime)
	
	return result
}


//Analyse le message
func msgAnalysis(input string) string {
    output := "Désolé, je n'ai pas reconnu la commande"
    switch input {
    	case "/start": output = "Bonjour Maître, que puis-je pour vous aujourd'hui?"
    	case "/help": output = "Je m'appelle Goule, et je sers la maison de mon Maître"
    	case "/temp": output = getTemperatures()
    	case "/conso": output = getConsoElectrique()
    	case "/internet": output = getInternet()
    	case "/traffic": output = getTraffic()
    	default: output = "Désolé, je n'ai pas reconnu la commande"
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
            //log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)
            input := update.Message.Text
            output := msgAnalysis(input)
        	msg := tgbotapi.NewMessage(update.Message.Chat.ID, output)
        	msg.ParseMode = "Markdown" 
        	bot.Send(msg)
        } else {
        	//log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)
        	msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Désolé je n'obéis qu'à mon Maître")
        	bot.Send(msg)
        }

    }
}