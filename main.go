package main

import (
	"encoding/json"
	"log"
	"math"
	"math/rand"
	"os"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/spf13/viper"
	"os/signal"
	"syscall"
)

// StatusData 구조체 정의
type StatusData struct {
	Tyield         float64 `json:"Tyield"`
	Dyield         float64 `json:"Dyield"`
	PF             float64 `json:"PF"`
	Pmax           float64 `json:"Pmax"`
	Pac            float64 `json:"Pac"`
	Sac            float64 `json:"Sac"`
	Uab            float64 `json:"Uab"`
	Ubc            float64 `json:"Ubc"`
	Uca            float64 `json:"Uca"`
	Ia             float64 `json:"Ia"`
	Ib             float64 `json:"Ib"`
	Ic             float64 `json:"Ic"`
	Freq           float64 `json:"Freq"`
	Tmod           float64 `json:"Tmod"`
	Tamb           float64 `json:"Tamb"`
	Mode           string  `json:"Mode"`
	Qac            int     `json:"Qac"`
	BusCapacitance float64 `json:"BusCapacitance"`
	AcCapacitance  float64 `json:"AcCapacitance"`
	Pdc            float64 `json:"Pdc"`
	PmaxLim        float64 `json:"PmaxLim"`
	SmaxLim        float64 `json:"SmaxLim"`
}

// SensorData 구조체 정의
type SensorData struct {
	Device    string     `json:"Device"`
	Timestamp string     `json:"Timestamp"`
	ProVer    int        `json:"ProVer"`
	MinorVer  int        `json:"MinorVer"`
	SN        int64      `json:"SN"`
	Model     string     `json:"model"`
	Status    StatusData `json:"Status"`
}

const (
	proVerVal   = 12345
	minorVerVal = 12345
	snVal       = 123456789012345
)

// roundToOneDecimal 함수
func roundToOneDecimal(value float64) float64 {
	return math.Round(value*10) / 10
}

// generateMode 함수
func generateMode() string {
	r := rand.Float64() * 100
	switch {
	case r < 1:
		return "Fault"
	case r < 3:
		return "Check"
	case r < 13:
		return "Standby"
	case r < 99:
		return "Running"
	default:
		return "Derate"
	}
}

// generateRandomFloat 함수
func generateRandomFloat(min, max float64) float64 {
	return roundToOneDecimal(min + rand.Float64()*(max-min))
}

// GenerateData 함수
func GenerateData() SensorData {
	return SensorData{
		Device:    "IoT_002",
		Timestamp: time.Now().Format("2006-01-02 15:04:05"),
		ProVer:    proVerVal,
		MinorVer:  minorVerVal,
		SN:        snVal,
		Model:     "mo-2093",
		Status: StatusData{
			Tyield:         generateRandomFloat(0, 5000),
			Dyield:         generateRandomFloat(0, 500),
			PF:             generateRandomFloat(0, 100),
			Pmax:           generateRandomFloat(0.1, 1000),
			Pac:            generateRandomFloat(0.1, 1000),
			Sac:            generateRandomFloat(0.1, 1000),
			Uab:            generateRandomFloat(0.1, 1000),
			Ubc:            generateRandomFloat(0.1, 1000),
			Uca:            generateRandomFloat(0.1, 1000),
			Ia:             generateRandomFloat(0.1, 100),
			Ib:             generateRandomFloat(0.1, 100),
			Ic:             generateRandomFloat(0.1, 100),
			Freq:           generateRandomFloat(0.1, 1000),
			Tmod:           generateRandomFloat(0.1, 1000),
			Tamb:           generateRandomFloat(0.1, 1000),
			Mode:           generateMode(),
			Qac:            rand.Intn(2000),
			BusCapacitance: generateRandomFloat(0, 20),
			AcCapacitance:  generateRandomFloat(0, 20),
			Pdc:            generateRandomFloat(0, 50),
			PmaxLim:        generateRandomFloat(0, 50),
			SmaxLim:        generateRandomFloat(0, 50),
		},
	}
}

func main() {
	// 랜덤 시드 초기화
	rand.Seed(time.Now().UnixNano())

	// 개발 환경에서 .env 파일을 사용하려면 주석을 해제하세요.
	// godotenv.Load()

	// PROFILE 환경 변수 확인
	profile := os.Getenv("PROFILE")
	isProd := strings.ToLower(profile) == "prod"

	// Viper 설정
	viper.SetConfigName("config") // 파일 이름 (확장자 제외)
	viper.SetConfigType("json")   // 파일 형식
	viper.AddConfigPath(".")      // 파일 경로

	// 프로덕션이 아닌 경우 config.json을 읽습니다.
	if !isProd {
		if err := viper.ReadInConfig(); err != nil {
			log.Fatalf("Failed to read config file: %v", err)
		}
	} else {
		// 프로덕션인 경우 환경 변수를 읽습니다.
		viper.AutomaticEnv()
		viper.SetEnvPrefix("mqtt") // 환경 변수 프리픽스 설정 (예: MQTT_BROKER)
		viper.BindEnv("broker")
		viper.BindEnv("topic")
		viper.BindEnv("client_id")
		viper.BindEnv("username")
		viper.BindEnv("password")

		viper.BindEnv("producer.publish_interval_seconds")
	}

	// MQTT 설정 읽기
	mqttBroker := getMQTTBroker(isProd)
	topic := viper.GetString("mqtt.topic")
	clientID := viper.GetString("mqtt.client_id")
	username := viper.GetString("mqtt.username")
	password := viper.GetString("mqtt.password")

	// MQTT 설정 정보 로그 출력
	log.Println("=== MQTT 설정 정보 ===")
	log.Printf("Broker: %s", mqttBroker)
	log.Printf("Topic: %s", topic)
	log.Printf("Client ID: %s", clientID)
	if username != "" {
		log.Printf("Username: %s", username)
	} else {
		log.Println("Username: (not set)")
	}
	log.Println("======================")

	// MQTT 클라이언트 옵션 설정
	opts := mqtt.NewClientOptions()
	opts.AddBroker(mqttBroker)
	opts.SetClientID(clientID)

	// SASL 인증이 필요한 경우 설정
	if username != "" && password != "" {
		opts.SetUsername(username)
		opts.SetPassword(password)
	}

	// MQTT 클라이언트 생성
	client := mqtt.NewClient(opts)

	// MQTT 클라이언트 연결
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatalf("MQTT 연결 실패: %v", token.Error())
	}
	defer client.Disconnect(250)

	// Graceful Shutdown을 위한 채널 설정
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	// Graceful Shutdown을 위한 goroutine
	done := make(chan bool, 1)
	go func() {
		sig := <-sigs
		log.Printf("Received signal %s, shutting down gracefully...", sig)
		client.Disconnect(250)
		done <- true
	}()

	// 데이터 전송 주기 설정 (config.json 또는 환경 변수에서 읽어오기)
	intervalSeconds := viper.GetInt("producer.publish_interval_seconds")
	ticker := time.NewTicker(time.Duration(intervalSeconds) * time.Second)
	defer ticker.Stop()

	log.Println("MQTT Producer가 시작되었습니다.")

	for {
		select {
		case <-ticker.C:
			// 데이터 생성
			data := GenerateData()

			// JSON 직렬화
			payload, err := json.Marshal(data)
			if err != nil {
				log.Printf("JSON 직렬화 실패: %v", err)
				continue
			}

			// MQTT 메시지 전송
			token := client.Publish(topic, 0, false, payload)
			token.Wait()
			if token.Error() != nil {
				log.Printf("메시지 전송 실패: %v", token.Error())
			} else {
				log.Printf("메시지 전송 성공: %s", payload)
			}
		case <-done:
			log.Println("프로그램을 종료합니다.")
			return
		}
	}
}

// getMQTTBroker 함수
func getMQTTBroker(isProd bool) string {
	if isProd {
		// 프로덕션 환경에서는 환경 변수에서 브로커 주소를 읽습니다.
		broker := os.Getenv("MQTT_BROKER")
		if broker == "" {
			log.Fatal("프로덕션 환경에서 MQTT_BROKER 환경 변수가 설정되지 않았습니다.")
		}
		return broker
	}
	// 프로덕션 환경이 아닌 경우 config.json에서 브로커 주소를 읽습니다.
	return viper.GetString("mqtt.broker")
}
