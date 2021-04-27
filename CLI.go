package main

import (
	"fmt"
	"strconv"
	"strings"
)

type URL struct {
	Base string
	Appendages []string
}

type Time struct {
	Hours int
	Minutes int
	Seconds int
}

type Hammer struct {
	url URL
	time Time
	perSecond int
}

func takeURLInfo()URL{
	fmt.Println("Enter the base of the URL to test: ")
	var urlName string
	fmt.Scanln(&urlName)
	fmt.Println("Enter the API extensions to test, separated by commas: ")
	var csvURLs string
	fmt.Scanln(&csvURLs)
	separated := strings.Split(csvURLs,",")
	return URL{
		Base:       urlName,
		Appendages: separated,
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

func testForHowLong(url URL)Hammer{
	fmt.Println("Enter how long you want to test for in the form of HH:MM:SS: ")
	var time string
	fmt.Scanln(&time)
	times := strings.Split(time,":")
	hours := toIntZeroIfFail(times[0],-1)
	minutes := toIntZeroIfFail(times[1],60)
	seconds := toIntZeroIfFail(times[2],60)
	limitTime := Time{
		Hours:   hours,
		Minutes: minutes,
		Seconds: seconds,
	}
	fmt.Println("Enter how many requests per second you would like to make: ")
	var perSecondStr string
	fmt.Scanln(&perSecondStr)
	perSecond := toIntZeroIfFail(perSecondStr,-1)
	return Hammer{
		url:       url,
		time:      limitTime,
		perSecond: perSecond,
	}

}




