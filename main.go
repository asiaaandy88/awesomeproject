package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

type configJson struct {
	NodeID             string      `json:"node_id"`
	Url                string      `json:"url"`
	Term               int         `json:"term"`
	CycleTime          string      `json:"cycle_time"`
	InputDelay         string      `json:"input_delay"`
	TargetAccumulate   int         `json:"target_accumulate"`
	ActualAccumulate   int         `json:"actual_accumulate"`
	AutoReset          []string    `json:"auto_reset"`
	ShiftTimeMonday    []ShiftTime `json:"shift_monday"`
	ShiftTimeTuesday   []ShiftTime `json:"shift_tuesday"`
	ShiftTimeWednesday []ShiftTime `json:"shift_wednesday"`
	ShiftTimeThursday  []ShiftTime `json:"shift_thursday"`
	ShiftTimeFriday    []ShiftTime `json:"shift_friday"`
	ShiftTimeSaturday  []ShiftTime `json:"shift_saturday"`
	ShiftTimeSunday    []ShiftTime `json:"shift_sunday"`
}

type ShiftTime struct {
	OnTime  string `json:"on_time"`
	OffTime string `json:"off_time"`
}

type CounterData struct {
	Plan   int `json:"plan"`
	Target int `json:"target"`
	Actual int `json:"actual"`
}

func httpConnect(url string, data *CounterData, config *configJson) {
	//Comment
	fmt.Println("URL:>", url)

	tr := &http.Transport{
		MaxIdleConns:       10,
		IdleConnTimeout:    30 * time.Second,
		DisableCompression: true,
	}
	reconnectCount := 1

	currentDate := time.Now().Format("2006-01-02")
	fmt.Println(currentDate)
	fileWriteData, err := os.OpenFile("./log/"+currentDate+".txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
	for {
		fmt.Println("Connect# ", reconnectCount)

		client := &http.Client{Transport: tr}
		for {
			currentTime := time.Now()
			iDiff := data.Target - data.Actual

			tempDate := currentTime.Format("2006-01-02")
			if tempDate != currentDate {
				err = fileWriteData.Close()
				if err != nil {
					panic(err)
				}
				currentDate = time.Now().Format("2006-01-02")
				fileWriteData, err = os.OpenFile("./log/"+currentDate+".txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
				if err != nil {
					panic(err)
				}

			}
			logWrite := fmt.Sprintf("%s %d %d %d %d\n", currentTime.Format("2006-01-02 15:04:05.000"), data.Plan, data.Target, data.Actual, iDiff)
			if _, err := fileWriteData.WriteString(logWrite); err != nil {
				log.Println(err)
			}

			values := fmt.Sprintf("{\"plan\":%d,\"target\":%d,\"actual\":%d, \"term\":%d}", data.Plan, data.Target, data.Actual, config.Term)
			jsonStr := []byte(values)
			resp, err := client.Post(url, "application/json", bytes.NewBuffer(jsonStr))
			if err != nil {
				fmt.Println(err)
				break
			}

			reconnectCount = 1
			//fmt.Println("response Status:", resp.Status)
			//fmt.Println("response Headers:", resp.Header)
			body, _ := ioutil.ReadAll(resp.Body)
			//fmt.Println("response Body:", string(body))
			if string(body) != "{}" {
				counterData := CounterData{}
				_ = json.Unmarshal(body, &counterData)
				data.Plan = counterData.Plan
				data.Target = counterData.Target
				data.Actual = counterData.Actual

				err = json.Unmarshal(body, &config)
				if err != nil {
					panic(err)
				}
				jsonString, _ := json.Marshal(config)
				ioutil.WriteFile("config.json", jsonString, os.ModePerm)
				fmt.Println("data changed")
			}
			time.Sleep(1000 * time.Millisecond)
		}
		client.CloseIdleConnections()
		time.Sleep(time.Duration(math.Exp2(float64(reconnectCount))) * time.Second)
		if reconnectCount < 9 {
			reconnectCount++
		}

	}

}

func inTimeSpan(start, end, check time.Time) bool {
	return check.After(start) && check.Before(end)
}

func convertTime(OnOffTime string, currentRFC3339 string) time.Time {
	prefix := currentRFC3339[0:11]
	postfix := currentRFC3339[19:]
	timeConverted, err := time.Parse(time.RFC3339, prefix+OnOffTime+postfix)
	if err != nil {
		panic(err)
	}
	return timeConverted
}

func targetCount(data *CounterData, config *configJson) {
	var shiftTime []ShiftTime
	for {
		currentTime := time.Now()
		weekday := int(currentTime.Weekday())
		currentTimeFormat := currentTime.Format(time.RFC3339)
		switch weekday {
		case 1:
			shiftTime = config.ShiftTimeMonday
		case 2:
			shiftTime = config.ShiftTimeTuesday
		case 3:
			shiftTime = config.ShiftTimeWednesday
		case 4:
			shiftTime = config.ShiftTimeThursday
		case 5:
			shiftTime = config.ShiftTimeFriday
		case 6:
			shiftTime = config.ShiftTimeSaturday
		case 0:
			shiftTime = config.ShiftTimeSunday
		}

		for i := 0; i < len(shiftTime); i++ {
			onTime := convertTime(shiftTime[i].OnTime, currentTimeFormat)
			offTime := convertTime(shiftTime[i].OffTime, currentTimeFormat)
			if inTimeSpan(onTime, offTime, currentTime) {
				data.Target += config.TargetAccumulate
			}
		}

		//currentTimeFormat := currentTime.Format(time.RFC3339)

		fCycleTime, err := strconv.ParseFloat(config.CycleTime, 32)
		if err != nil {
			panic(err)
		}
		fCycleTime *= 1000
		time.Sleep(time.Duration(fCycleTime) * time.Millisecond)
	}
}

func targettest(data *CounterData, config *configJson) {
	fmt.Println(config.ShiftTimeMonday[2].OnTime)
	fmt.Println(config.ShiftTimeMonday[2].OffTime)
	//t, _ := time.Parse(, "01 Jan 15 " + config.ShiftTimeMonday[0].OffTime)
	currentTime := time.Now()
	currentTimeFormat := currentTime.Format(time.RFC3339)
	onTime := convertTime(config.ShiftTimeMonday[0].OnTime, currentTimeFormat)
	offTime := convertTime(config.ShiftTimeMonday[0].OffTime, currentTimeFormat)
	fmt.Println(onTime)
	fmt.Println(offTime)
	fmt.Println(int(onTime.Weekday()))
	//fmt.Println(currentTime.Format(time.RFC3339))
	//fmt.Println(currentTime.Format(time.RFC3339)[0:11])
	//fmt.Println(currentTime.Format(time.RFC3339)[19:])
	fmt.Println(inTimeSpan(onTime, offTime, currentTime))

	time.Sleep(1000 * time.Millisecond)
}

func readLastLineData() [3]int {
	currentDate := time.Now().Format("2006-01-02")
	fName := "./log/" + currentDate + ".txt"
	file, err := os.Open(fName)
	if err != nil {
		dataValues := [3]int{0, 0, 0}
		return dataValues
	}
	defer file.Close()

	stat, _ := os.Stat(fName)

	var readFileBufferSize int64
	var start int64
	var buf []byte
	readFileBufferSize = 1024

	start = stat.Size() - readFileBufferSize
	buf = make([]byte, readFileBufferSize)

	if start < 0 {
		start = 0
		buf = make([]byte, stat.Size())
	}

	_, err = file.ReadAt(buf, start)
	if err != nil {
		panic(err)
	}
	lines := strings.Split(string(buf), "\n")

	planValues := 0
	targetValues := 0
	actualValues := 0

	for i := len(lines) - 1; i >= 0; i-- {
		fileTime := strings.Split(lines[i], " ")
		if len(fileTime) != 6 {
			continue
		}
		planValues, err = strconv.Atoi(fileTime[2])
		if err != nil {
			continue
		}
		targetValues, err = strconv.Atoi(fileTime[3])
		if err != nil {
			continue
		}
		actualValues, err = strconv.Atoi(fileTime[4])
		if err != nil {
			continue
		}
		break
	}

	dataValues := [3]int{planValues, targetValues, actualValues}
	return dataValues
}

func main() {
	var config configJson
	_ = os.Mkdir("log", os.ModePerm)

	content, err := ioutil.ReadFile("config.json")
	if err != nil {
		log.Fatal(err)
	}

	err = json.Unmarshal([]byte(content), &config)
	if err != nil {
		log.Fatal(err)
	}

	dataValues := readLastLineData()

	data := CounterData{
		Plan:   dataValues[0],
		Target: dataValues[1],
		Actual: dataValues[2],
	}

	go httpConnect(config.Url+config.NodeID+"/", &data, &config)
	go targetCount(&data, &config)

	for {
		data.Plan += 1
		data.Actual += 1
		time.Sleep(1000 * time.Millisecond)
	}

}
