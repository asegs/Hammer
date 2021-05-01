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
	runtime time.Duration
	pending int

}

var outboundReqs = make([]int,0)
var hammers []Hammer
var s1 = rand.NewSource(time.Now().UnixNano())
var r1 = rand.New(s1)
var closures [][]TimeClosure
var countChannel = make([]chan int,0)
var activeHammers []Hammer
var averageTimeClosures [][]TimeClosure

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
	return fmt.Sprintf("Name: %s, Base URL: %s, runtime: %s, %d requests/second, %d variable requests\n",hammer.name,hammer.url.Base,timeToString(hammer.time),hammer.perSecond,len(hammer.url.Appendages))
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
	closures[outboundIdx] = append(closures[outboundIdx],TimeClosure{
		URL:   URL,
		Type:  reqType,
		runtime: end.Sub(start),
		pending: outboundReqs[outboundIdx],

	})
	countChannel[outboundIdx]<--1
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

func runHammer(index int,outboundIndex int){
	h := activeHammers[index]
	for i:=0;i<timeToSeconds(h.time);i++{
		hammer := activeHammers[index]
		waitTime := int(1.0/float64(hammer.perSecond)*1000)
		for b:=0;b<hammer.perSecond;b++{
			fullURL := ""
			typeOf := ""
			if len(hammer.url.Appendages)== 0{
				fullURL = hammer.url.Base
				typeOf = "GET"
			}else{
				idx := r1.Intn(len(hammer.url.Appendages))
				fullURL = hammer.url.Base+hammer.url.Appendages[idx].Ext
				typeOf = strings.ToUpper(hammer.url.Appendages[idx].Type)
			}

			go makeTypedRequest(fullURL,typeOf,outboundIndex)
			time.Sleep(time.Duration(waitTime)*time.Millisecond)
		}
	}
}

func outboundWatcher(index int){
	for true{
		change := <- countChannel[index]
		outboundReqs[index]+=change
	}
}

func sumClosureTime(t []TimeClosure,startPercent int, endPercent int)time.Duration{
	totalTime := time.Duration(0.0)
	start := int(float64(startPercent)/100*float64(len(t)))
	end :=  int(float64(endPercent)/100*float64(len(t)))
	for i:=start;i<end;i++ {
		totalTime+=t[i].runtime
	}
	if end-start == 0{
		end+=1
	}
	return totalTime / time.Duration(end-start)
}

func viewActiveHammers(){
	for i,h := range activeHammers{
		fmt.Println(len(averageTimeClosures[i]))
		fmt.Printf("Hammering %s %d times per second\n%d requests made, %d requests pending\nAverage response time: %v\nAverage response time (first 10%%): %v\nAverage response time (latest 10%%): %v\nFirst response time: %v  Latest response time: %v\n\n",h.url.Base,h.perSecond,len(averageTimeClosures[i])*(h.perSecond/2),outboundReqs[i],sumClosureTime(averageTimeClosures[i],0,100),sumClosureTime(averageTimeClosures[i],0,10),sumClosureTime(averageTimeClosures[i],90,100),averageTimeClosures[i][0].runtime,averageTimeClosures[i][len(averageTimeClosures[i])-1].runtime)
	}
}

func setNewHammerRate(){
	for i,h := range activeHammers{
		fmt.Printf("%d.) Name: %s  Hammers per second: %d\n",i,h.name,h.perSecond)
	}
	fmt.Println("Enter the index of the hammer to modify, followed by the new per second rate (ex. 2 150) or press enter to not change anything:")
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	text := scanner.Text()
	if text == ""{
		return
	}else{
		tokens := strings.Split(text," ")
		if len(tokens)!= 2{
			fmt.Println("Invalid format")
			return
		}
		index,err := strconv.Atoi(tokens[0])
		if err != nil{
			fmt.Println("Invalid index")
			return
		}
		perSecond,err := strconv.Atoi(tokens[1])
		if err != nil{
			fmt.Println("Invalid index")
			return
		}
		activeHammers[index].perSecond = perSecond
	}

}

func console(){
	for true {
		fmt.Println("Would you like to create a hammer (c), run a hammer (r), view active hammers (v), set hammer rate (s), or kill all hammers and quit (q)?: ")
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
			activeHammers = append(activeHammers,hammers[index])
			closures = append(closures,make([]TimeClosure,0))
			averageTimeClosures = append(averageTimeClosures,make([]TimeClosure,0))
			go runHammer(index,len(outboundReqs)-1)
			go outboundWatcher(len(outboundReqs)-1)
			go offloader(index)
			fmt.Println("Started hammering: "+hammers[index].url.Base)
		}else if response == "v"{
			viewActiveHammers()
		}else if response=="s"{
			setNewHammerRate()
		}else if response == "q"{
			logAllTimeClosures()
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

func offload(count int,index int){
	if len(closures[index])>count{
		toOffload := closures[index][0:count]
		closures[index] = closures[index][count:]
		filename := nameActiveHammer(index)
		sb := strings.Builder{}
		if _, err := os.Stat(filename); !os.IsNotExist(err){
			sb.WriteString(ReadToString(filename)+"\n")
		}else{
			sb.WriteString(fmt.Sprintf("Name: %s\nBase URL: %s\nRequests per second: %d\n\n\n\n",activeHammers[index].name,activeHammers[index].url.Base,activeHammers[index].perSecond))
		}
		avg := sumClosureTime(toOffload,0,100)
		sb.WriteString(fmt.Sprintf("%v\n",time.Now()))
		sb.WriteString(fmt.Sprintf("Per second: %d, average time: %v\n",count*2,avg))
		averageTimeClosures[index] = append(averageTimeClosures[index],TimeClosure{
			URL:     activeHammers[index].url.Base,
			Type:    "GET",
			runtime: avg,
			pending: outboundReqs[index],
		})
		Write(filename,sb.String())
	}

}

func offloader(index int){
	for true{
		offload(activeHammers[index].perSecond/2,index)
		time.Sleep(30*time.Millisecond)
	}
}


func timeClosureArrToText(t []TimeClosure)string{
	sb := strings.Builder{}
	totalTime := time.Duration(0)
	for _,tc := range t{
		sb.WriteString(fmt.Sprintf("URL: %s    Time: %v    Pending Requests: %d\n",tc.URL,tc.runtime,tc.pending))
		totalTime+=tc.runtime
	}
	body := fmt.Sprintf("Average time: %v\n\n\n\n%s",totalTime/time.Duration(len(t)),sb.String())
	return body
}

func nameActiveHammer(i int)string{
	return fmt.Sprintf("files/%s.txt",activeHammers[i].name)
}

func logAllTimeClosures(){
	for i,_ := range closures{
		offload(len(closures[i]),i)
		//fileName := nameActiveHammer(i)
		//body := timeClosureArrToText(list)
		//fmtBody := fmt.Sprintf("Name: %s\nBase URL: %s\nRequests per second: %d\n%s",activeHammers[i].name,activeHammers[i].url.Base,activeHammers[i].perSecond,body)
		//Write(fileName,fmtBody)
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


/*
Bigger goals:

Hashmap of unique URLs, some take longer than others

Allow uploading of JSON files to an API

Record errors

Show average start time first ~10% of requests and average end time last ~10% of requests difference (in file)

GUI ;)
*/
