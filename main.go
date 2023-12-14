package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/streadway/amqp"
	"github.com/twilio/twilio-go"
	twilioApi "github.com/twilio/twilio-go/rest/api/v2010"
	"sms-consumer/config"
)

// Define a global variable to hold the RabbitMQ connection
var rabbitMQConn *amqp.Connection
var rabbitMQConnMutex sync.Mutex

func connectToRabbitMQ() (*amqp.Connection, error) {
	// Retrieve RabbitMQ connection URL from environment variables for security
	rabbitMQURL := config.ReadEnv("RABBITMQ_URL")
	// rabbitMQURL := "amqp://guest:guest@localhost:5672/" // local rabbitmq url
	if rabbitMQURL == "" {
		log.Fatal("RABBITMQ_URL environment variable not set")
	}

	// Establish a connection to RabbitMQ
	conn, err := amqp.Dial(rabbitMQURL)
	if err != nil {
		return nil, err
	}

	fmt.Println("Connected to RabbitMQ successfully")

	return conn, nil
}

func declareQueue(ch *amqp.Channel) error {
	queueName := "SMS_QUEUE"

	_, err := ch.QueueDeclare(queueName, false, false, false, false, nil)
	if err != nil {
		return err
	}

	fmt.Println("Queue declared successfully")
	return nil
}

func consumeSMSRequests(conn *amqp.Connection, client *twilio.RestClient) {
	// Create a channel
	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("Failed to open a channel: %v", err)
	}
	defer ch.Close()

	// Declare the queue
	if err := declareQueue(ch); err != nil {
		log.Fatalf("Failed to declare the queue: %v", err)
	}

	queueName := "SMS_QUEUE"

	msgs, err := ch.Consume(
		queueName,
		"",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatalf("Failed to consume messages from the queue: %v", err)
	}

	for msg := range msgs {
		// Process SMS requests here
		messageContent := string(msg.Body)

		fmt.Println(messageContent)

		fromPhoneNumber := os.Getenv("TWILIO_PHONE_NUMBER")
		if fromPhoneNumber == "" {
			fromPhoneNumber = "+18886880943" // Set your default fallback number
		}
		fmt.Printf("FROM PHONE NUMBER: %s\n", fromPhoneNumber)

		toPhoneNumber := os.Getenv("RECIPIENT_PHONE_NUMBER")
		if toPhoneNumber == "" {
			toPhoneNumber = "+18777804236" // Set your default fallback recipient number
		}
		fmt.Printf("TO PHONE NUMBER: %s\n", toPhoneNumber)

		params := &twilioApi.CreateMessageParams{}
		params.SetTo(toPhoneNumber)
		params.SetFrom(fromPhoneNumber)
		params.SetBody(messageContent)

		resp, err := client.Api.CreateMessage(params)
		if err != nil {
			fmt.Println("Error sending SMS message: " + err.Error())
		} else {
			response, _ := json.Marshal(*resp)
			fmt.Println("Response: " + string(response))
		}
	}
}

func main() {
	// ... Initialize any other necessary components ...
	// Initialize RabbitMQ connection in the main function
	var err error
	rabbitMQConn, err = connectToRabbitMQ()
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	defer rabbitMQConn.Close()

	TWILIO_SID := os.Getenv("TWILIO_SID")
	TWILIO_AUTH_TOKEN := os.Getenv("TWILIO_AUTH_TOKEN")
	fmt.Printf("TWILIO SID: %s\n", TWILIO_SID)
	fmt.Printf("TWILIO AUTH TOKEN: %s\n", TWILIO_AUTH_TOKEN)

	// Check if environment variables are empty or missing
	if TWILIO_SID == "" || TWILIO_AUTH_TOKEN == "" {
		fmt.Println("TWILIO_SID or TWILIO_AUTH_TOKEN environment variables are missing or empty.")
		return
	}

	// Initialize Twilio client
	client := twilio.NewRestClientWithParams(twilio.ClientParams{
		Username: os.Getenv("TWILIO_SID"),
		Password: os.Getenv("TWILIO_AUTH_TOKEN"),
	})

	// Start the SMS request consumer
	consumeSMSRequests(rabbitMQConn, client)
}
