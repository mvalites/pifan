/*

Summary: toy fan controller for my pi 4.
License: Copyright 2020 Matt Valites; See LICENSE
Docs: See readme.md

*/

package main

import (
	"log"
	"fmt"
	"time"
	"os"

	"github.com/warthog618/gpiod"
	"github.com/warthog618/gpiod/device/rpi"
)


// global config oject
type FanCTX struct {
	temp_file	string		/* location of the sysfs file */
	fan_on_temp	int		/* Temp to turn fan on: degrees C * 1000 */
	sample_interval time.Duration  /* sleep between samples when checking above, (nanoseconds) */
	gpio_line	* gpiod.Line	/* the "line" (pin) to which the fan is plugged into */
	next_change	time.Time	/* some smoothing is nice */
}



func getTemp(path string) int {
	file, err := os.Open(path)
	if err != nil {
		log.Fatalf("Cannot open temp file: %s [%s]", path, err )
	}
	defer file.Close()
	value := 0
	n, err := fmt.Fscanf(file, "%d", &value)
	if n != 1 {
		log.Fatalf("Failed to parse temperature: %s [%s]", path, err )
	}
	return value
}

func fanLoop(ctx *FanCTX) {
	/* monitor and control a fan */

	for {
		current_temp := getTemp(ctx.temp_file)
		val:= -1
		now := time.Now()
		if (current_temp > ctx.fan_on_temp) {
			val = 1 // always on if over temp
		} else if now.After(ctx.next_change) {
			val = 0 // off only if we've been on long enough
		}
		if val >= 0 {
			ctx.gpio_line.SetValue(val)
			ctx.next_change = now.Add(ctx.sample_interval * 5)
		}
		/* go to sleep */
		log.Printf("Setting FAN(%d) current_temp: %d, cut_off: %d, next_change: %s" ,
									val,
									current_temp,
									ctx.fan_on_temp,
									ctx.next_change.String())
		time.Sleep(ctx.sample_interval)
	}
}

func main() {
	/* TODO: 
	 - when shutting down how do we break the loop nicely
	 - add some smoothing, honestly the sample time probably is enough
	 - we could support PWM at some poin!
	 - read config file, maybe re-load config on change? 
	*/


	/* setup the GPIO stuff */
	/* TODO: I don't fully understand chip0 vs chip1, both currently exist */
	chip, err := gpiod.NewChip("gpiochip0", gpiod.WithConsumer("pifan"))
	if err != nil {
		log.Fatal(err)
	}
	/* get line */
	line, err := chip.RequestLine(rpi.GPIO17, gpiod.AsOutput(1))
	if err != nil {
		log.Fatal(err)
	}
	
	g_config := FanCTX{ temp_file: "/sys/class/thermal/thermal_zone0/temp",
		       fan_on_temp: 70 * 1000,
		       sample_interval: 1000 * time.Millisecond,
		       gpio_line: line,
		       next_change : time.Now() }

	log.Print("CONFIG: ", fmt.Sprintf("%v", g_config))

	fanLoop(&g_config)
}

