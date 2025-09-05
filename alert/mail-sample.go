package alert

import (
	"fmt"
	"os"

	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
)

func SendSampleMail() {
	from := mail.NewEmail("Alert Manager", "alerts@em2695.tufwgo.store")
	subject := "[TUFWGo] Rule Added - Allow TCP 22 from 192.168.1.1"
	to := mail.NewEmail("Rohit Gurunathan", "rohitgurunathan@gmil.com")
	plainTextContent := `Hello,a
	An action was performed on your firewall via TUFWGo.
	📌 Action: Rule Added
	📌 Status: Success
	📌 Timestamp: 2024-10-05 14:30:00
	📌 Executed By: root
	📌 Hostname: raspberrypi
	📌 Rule Details:
		- Action: Allow
		- Direction: Inbound
		- From: 192.168.1.1
		- To: Any
		- Port: 22
		- Protocol: TCP
		- App Profile: N/A

	🏷️ Command Executed:
		ufw allow from 192.168.1.5 to any port 22 proto tcp

	TUFWGo Alert Manager
	`
	htmlContent := fmt.Sprintf("<strong>%s</strong>", plainTextContent)
	message := mail.NewSingleEmail(from, subject, to, plainTextContent, htmlContent)
	client := sendgrid.NewSendClient(os.Getenv("SENDGRID_API_KEY"))
	response, err := client.Send(message)
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(response.StatusCode)
		fmt.Println(response.Body)
		fmt.Println(response.Headers)
	}
}
