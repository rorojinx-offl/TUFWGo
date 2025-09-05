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
	to := mail.NewEmail("Rohit Gurunathan", "rohitgurunathan@gmail.com")
	plainTextContent := `Hello,\n
	An action was performed on your firewall via TUFWGo.\n
	📌 Action: Rule Added\n
	📌 Status: Success\n
	📌 Timestamp: 2024-10-05 14:30:00\n
	📌 Executed By: root\n
	📌 Hostname: raspberrypi\n
	📌 Rule Details:\n\t
		- Action: Allow\n\t
		- Direction: Inbound\n\t
		- From: 192.168.1.1\n\t
		- To: Any\n\t
		- Port: 22\n\t
		- Protocol: TCP\n\t
		- App Profile: N/A\n\n

	🏷️ Command Executed:\n\t
		ufw allow from 192.168.1.5 to any port 22 proto tcp\n\n

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
