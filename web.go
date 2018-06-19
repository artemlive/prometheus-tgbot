package main

import (
	"log"
	"github.com/ttacon/emoji"
	"github.com/go-telegram-bot-api/telegram-bot-api"
	"sort"
	"fmt"
	"github.com/gin-gonic/gin"
	"strconv"
	"net/http"
	"strings"
	"bytes"
	"io"
	"github.com/gin-gonic/gin/binding"
	"encoding/json"
)

func GET_Handling(c *gin.Context) {
	log.Printf("Recived GET")
	chatid, err := strconv.ParseInt(c.Param("chatid"), 10, 64)
	if err != nil {
		log.Printf("Cat't parse chat id: %q", c.Param("chatid"))
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"err": fmt.Sprint(err),
		})
		return
	}

	log.Printf("Bot test: %d", chatid)
	msgtext := fmt.Sprintf("Some HTTP triggered notification by prometheus bot... %d", chatid)
	msg := tgbotapi.NewMessage(chatid, msgtext)
	sendmsg, err := bot.Send(msg)
	if err == nil {
		c.String(http.StatusOK, msgtext)
	} else {
		c.JSON(http.StatusBadRequest, gin.H{
			"err":     fmt.Sprint(err),
			"message": sendmsg,
		})
	}
}

func AlertFormatStandard(alerts Alerts) string {
	keys := make([]string, 0, len(alerts.GroupLabels))
	for k := range alerts.GroupLabels {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	groupLabels := make([]string, 0, len(alerts.GroupLabels))
	for _, k := range keys {
		groupLabels = append(groupLabels, fmt.Sprintf("%s=<code>%s</code>", k, alerts.GroupLabels[k]))
	}

	keys = make([]string, 0, len(alerts.CommonLabels))
	for k := range alerts.CommonLabels {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	commonLabels := make([]string, 0, len(alerts.CommonLabels))
	for _, k := range keys {
		if _, ok := alerts.GroupLabels[k]; !ok {
			commonLabels = append(commonLabels, fmt.Sprintf("%s=<code>%s</code>", k, alerts.CommonLabels[k]))
		}
	}

	keys = make([]string, 0, len(alerts.CommonAnnotations))
	for k := range alerts.CommonAnnotations {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	commonAnnotations := make([]string, 0, len(alerts.CommonAnnotations))
	for _, k := range keys {
		commonAnnotations = append(commonAnnotations, fmt.Sprintf("\n%s: <code>%s</code>", k, alerts.CommonAnnotations[k]))
	}

	alertDetails := make([]string, len(alerts.Alerts))
	for i, a := range alerts.Alerts {
		if instance, ok := a.Labels["instance"]; ok {
			instanceString, _ := instance.(string)
			alertDetails[i] += strings.Split(instanceString, ":")[0]
		}
		if job, ok := a.Labels["job"]; ok {
			alertDetails[i] += fmt.Sprintf("[%s]", job)
		}
		if a.GeneratorURL != "" {
			alertDetails[i] = fmt.Sprintf("<a href='%s'>%s</a>", a.GeneratorURL, alertDetails[i])
		}
	}
	return fmt.Sprintf(
		"<a href='%s/#/alerts?receiver=%s'>[%s:%d]</a>\ngrouped by: %s\nlabels: %s%s\n%s",
		alerts.ExternalURL,
		alerts.Receiver,
		strings.ToUpper(alerts.Status),
		len(alerts.Alerts),
		strings.Join(groupLabels, ", "),
		strings.Join(commonLabels, ", "),
		strings.Join(commonAnnotations, ""),
		strings.Join(alertDetails, ", "),
	)
}

func AlertFormatTemplate(alerts Alerts) string {
	var bytesBuff bytes.Buffer
	var err error

	writer := io.Writer(&bytesBuff)

	if *debug {
		log.Printf("Reloading Template\n")
		// reload template bacause we in debug mode
		tmpH = loadTemplate(cfg.TemplatePath)
	}

	tmpH.Funcs(funcMap)
	err = tmpH.Execute(writer, alerts)

	if err != nil {
		log.Fatalf("Problem with template execution: %v", err)
		panic(err)
	}

	return bytesBuff.String()
}

func POST_Handling(c *gin.Context) {
	var msgtext string
	var alerts Alerts
	var chatids []string
	if strings.Contains(c.Param("chatid"), string(',')) {
		log.Println("chatid countains multiply chat ids = " + c.Param("chatid"))
		chatids = strings.Split(c.Param("chatid"), ",")
		log.Printf("Chatids = %v", chatids)
	} else {
		chatid := c.Param("chatid")
		chatids = append(chatids, chatid)
	}

	for _, element := range chatids {
		log.Printf("chat id = %s", element)
		chatid, err := strconv.ParseInt(element, 10, 64)

		if err != nil {
			log.Printf("Cat't parse chat id: %q", c.Param("chatid"))
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"err": fmt.Sprint(err),
			})
			return
		}
		log.Printf("Bot alert post: %d", chatid)

		binding.JSON.Bind(c.Request, &alerts)

		s, err := json.Marshal(alerts)
		if err != nil {
			log.Print(err)
			return
		}

		log.Println("+------------------  A L E R T  J S O N  -------------------+")
		log.Printf("%s", s)
		log.Println("+-----------------------------------------------------------+\n\n")

		// Decide how format Text
		if cfg.TemplatePath == "" {
			msgtext = AlertFormatStandard(alerts)
		} else {
			msgtext = AlertFormatTemplate(alerts)
		}
		// Print in Log result message
		log.Println("+---------------  F I N A L   M E S S A G E  ---------------+")
		log.Println(msgtext)
		log.Println("+-----------------------------------------------------------+")
		log.Println(chatid)
		if len(msgtext) >= 4096 {
			msgtext = string(msgtext[0:4050])
			i := strings.LastIndex(msgtext, "<a")
			log.Println("Index: ", i)
			if i > -1 {
				msgtext = string(msgtext[0:i])
				msgtext += "... message was cut off (4096 chars limit)"
			}
		}
		msgtext = emoji.Emojitize(msgtext)
		msg := tgbotapi.NewMessage(chatid, msgtext)
		sizeOfCallbackData := len(fmt.Sprintf("am,120m,%s", alerts.GroupLabels["alertname"].(string)))
		log.Printf("size of callback data: %d", sizeOfCallbackData)
		if alerts.Status == "firing" {
			if sizeOfCallbackData < 64 {
				var silenceKeyboard = tgbotapi.NewInlineKeyboardMarkup(
					tgbotapi.NewInlineKeyboardRow(
						tgbotapi.NewInlineKeyboardButtonData("silence 30m", fmt.Sprintf("am,30m,%s", alerts.GroupLabels["alertname"].(string))),
						tgbotapi.NewInlineKeyboardButtonData("silence 60m", fmt.Sprintf("am,60m,%s", alerts.GroupLabels["alertname"].(string))),
						tgbotapi.NewInlineKeyboardButtonData("silence 120m", fmt.Sprintf("am,120m,%s", alerts.GroupLabels["alertname"].(string))),
					),
				)
				msg.ReplyMarkup = silenceKeyboard
			}
		}
		msg.ParseMode = tgbotapi.ModeHTML
		msg.DisableWebPagePreview = true
		sendmsg, err := bot.Send(msg)
		if err == nil {
			c.String(http.StatusOK, fmt.Sprintf("telegram msg sent to: %d.\n", chatid))
		} else {
			log.Printf("Error sending message: %s", err)
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"err":     fmt.Sprint(err),
				"message": sendmsg,
				"srcmsg":  fmt.Sprint(msgtext),
			})
			msg := tgbotapi.NewMessage(chatid, fmt.Sprintf("Error sending message: %s", err))
			bot.Send(msg)
		}
	}
}
