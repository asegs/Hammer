package main

import (
	"bufio"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

type URL struct {
	Base string
	Appendages []URLCombo
}

type fmtTime struct {
	Hours int
	Minutes int
	Seconds int
}

type Hammer struct {
	url URL
	time fmtTime
	perSecond int
	name string
}

type URLCombo struct {
	Ext string
	Type string
}

type TimeClosure struct {
	URL string
	Type string
	start time.Time
	end time.Time

}

var outboundReqs = make([]int,0)
var hammers []Hammer
var s1 = rand.NewSource(time.Now().UnixNano())
var r1 = rand.New(s1)
var closures []TimeClosure
var countChannel = make([]chan int,0)


func takeURLInfo()URL{
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("Enter the base of the URL to test: ")
	scanner.Scan()
	urlName := scanner.Text()
	fmt.Println("Enter the API extensions to test, one at a time.  Enter the request type (GET,POST..etc. afterwards with a space)  Press enter to finish: ")
	exts := make([]URLCombo,0)
	for true{
		fmt.Print(urlName)
		scanner.Scan()
		endpoint:= scanner.Text()
		if endpoint==""{
			break
		}
		parts := strings.Split(endpoint," ")
		exts = append(exts,URLCombo{
			Ext:  parts[0],
			Type: parts[1],
		})
		fmt.Printf("Enter the API extensions to test, one at a time.  Press enter to finish (%d added): \n",len(exts))
	}
	return URL{
		Base:       urlName,
		Appendages: exts,
	}
}

func intMin(i1 int,i2 int)int{
	if i1<i2{
		return i1
	}
	return i2
}

func toIntZeroIfFail(s string,maxVal int)int{
	i,err := strconv.Atoi(s)
	if err != nil{
		return 0
	}
	if maxVal==-1{
		return i
	}
	return intMin(maxVal,i)
}

func timeFromString(time string)fmtTime{
	times := strings.Split(time,":")
	hours := toIntZeroIfFail(times[0],-1)
	minutes := toIntZeroIfFail(times[1],60)
	seconds := toIntZeroIfFail(times[2],60)
	return fmtTime{
		Hours:   hours,
		Minutes: minutes,
		Seconds: seconds,
	}
}

func testForHowLong(url URL)Hammer{
	fmt.Println("Enter how long you want to test for in the form of HH:MM:SS: ")
	var time string
	fmt.Scanln(&time)
	limitTime := timeFromString(time)
	fmt.Println("Enter how many requests per second you would like to make: ")
	var perSecondStr string
	fmt.Scanln(&perSecondStr)
	perSecond := toIntZeroIfFail(perSecondStr,-1)
	fmt.Println("What is the name for this hammer?: ")
	var name string
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	name = scanner.Text()
	return Hammer{
		url:       url,
		time:      limitTime,
		perSecond: perSecond,
		name: name,
	}

}

func writeAllHammers(){
	sb := strings.Builder{}
	for _,hammer := range hammers{
		sb.WriteString(hammerToFileString(hammer)+"\n")
	}
	Write("files/hammers.txt",sb.String())
}

func createHammer()Hammer{
	url := takeURLInfo()
	return testForHowLong(url)
}

func timeToString(time fmtTime)string{
	return fmt.Sprintf("%d:%d:%d",time.Hours,time.Minutes,time.Seconds)
}

func hammerToFileString(hammer Hammer)string{
	sb := strings.Builder{}
	sb.WriteString(hammer.name+",")
	sb.WriteString(hammer.url.Base+",")
	sb.WriteString(timeToString(hammer.time)+",")
	sb.WriteString(strconv.Itoa(hammer.perSecond))
	for _,ext := range hammer.url.Appendages{
		sb.WriteString(","+ext.Ext+" "+ext.Type)
	}
	return sb.String()
}

func hammerToUserString(hammer Hammer)string{
	return fmt.Sprintf("Name: %s, Base URL: %s, runtime: %s, %d requests/seconds, %d variable requests\n",hammer.name,hammer.url.Base,timeToString(hammer.time),hammer.perSecond,len(hammer.url.Appendages))
}


func makeHammerLoop(){
	var input string
	for input!="q"{
		fmt.Println("Press any key to create a new hammer, or q to quit: ")
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Scan()
		input = scanner.Text()
		if input=="q"{
			return
		}
		hammers = append(hammers,createHammer())
		writeAllHammers()
	}
}

func displayHammers(){
	for i,hammer :=range hammers{
		fmt.Printf("%d. %s\n",i,hammerToUserString(hammer))
	}
}

func timeToSeconds(time fmtTime)int{
	return time.Hours*3600+time.Minutes*60+time.Seconds
}

func logCall(start time.Time,end time.Time,URL string,reqType string,outboundIdx int){
	closures = append(closures,TimeClosure{
		URL:   URL,
		Type:  reqType,
		start: start,
		end:   end,
	})
	countChannel[outboundIdx]<--1
	fmt.Println(URL)
	fmt.Println(end.Sub(start))
}

func makeTypedRequest(URL string,reqType string,outboundIdx int){
	//save errors, concurrent write to closures as channel
	start := time.Now()
	countChannel[outboundIdx]<-1
	if reqType == "GET"{
		_,_ = http.Get(URL)
		end := time.Now()
		logCall(start,end,URL,"GET",outboundIdx)
	}else if reqType == "POST"{
		_,_ = http.Post(URL,"application/json",nil)
		end := time.Now()
		logCall(start,end,URL,"POST",outboundIdx)
	}

}

func runHammer(hammer Hammer,outboundIndex int){
	waitTime := int(1.0/float64(hammer.perSecond)*1000)
	for i:=0;i<timeToSeconds(hammer.time);i++{
		for b:=0;b<hammer.perSecond;b++{
			idx := r1.Intn(len(hammer.url.Appendages))
			go makeTypedRequest(hammer.url.Base+hammer.url.Appendages[idx].Ext,strings.ToUpper(hammer.url.Appendages[idx].Type),outboundIndex)
			time.Sleep(time.Duration(waitTime)*time.Millisecond)
		}
	}
}

func outboundWatcher(index int){
	change := <- countChannel[index]
	outboundReqs[index]+=change
}

func console(){
	for true {
		fmt.Println("Would you like to create a hammer (c) run a hammer (r), or kill all hammers and quit (q)?: ")
		var response string
		fmt.Scanln(&response)
		if response == "c"{
			makeHammerLoop()
		}else if response == "r"{
			displayHammers()
			fmt.Println("Select a hammer by index: ")
			var choice string
			fmt.Scanln(&choice)
			index,err := strconv.Atoi(choice)
			if err != nil{
				continue
			}
			outboundReqs = append(outboundReqs,0)
			countChannel = append(countChannel,make(chan int,200))
			go runHammer(hammers[index],len(outboundReqs)-1)
		}else if response == "q"{
			//write all files first
			return
		}
	}
}

func textToHammer(text string)Hammer{
	csv := strings.Split(text,",")
	combos := make([]URLCombo,len(csv)-4)
	for i,word := range csv[4:]{
		sep := strings.Split(word," ")
		combos[i] = URLCombo{
			Ext:  sep[0],
			Type: sep[1],
		}
	}
	perSecond,err := strconv.Atoi(csv[3])
	if err != nil{
		perSecond = 1
	}
	return Hammer{
		url:       URL{
			Base:       csv[1],
			Appendages: combos,
		},
		time:      timeFromString(csv[2]),
		perSecond: perSecond,
		name: csv[0],
	}
}

func loadHammers(){
	fileText := ReadToString("files/hammers.txt")
	lines := strings.Split(fileText,"\n")
	for _,line := range lines{
		if len(strings.Split(line,","))<4{
			continue
		}
		hammers = append(hammers,textToHammer(line))
	}
}



func main(){
	loadHammers()
	console()
}


