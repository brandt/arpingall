package main

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"strings"
)

type Route struct {
	Interface   string
	Destination net.IP
	Gateway     net.IP
}

type iface struct {
	name string
	mac  string
	addr string
}

// Parse IP in format
func parseIP(str string) (net.IP, error) {
	bytes, err := hex.DecodeString(str)
	if err != nil {
		return nil, err
	}
	if len(bytes) != net.IPv4len {
		// TODO: IPv6 support
		return nil, fmt.Errorf("only IPv4 is supported")
	}
	bytes[0], bytes[1], bytes[2], bytes[3] = bytes[3], bytes[2], bytes[1], bytes[0]
	return net.IP(bytes), nil
}

func GetRoutes() ([]Route, error) {
	file, err := os.Open("/proc/net/route")
	if err != nil {
		log.Print("Can't open route file: ", err)
	}
	defer file.Close()

	routes := []Route{}

	scanner := bufio.NewReader(file)
	lineNum := 0
	for {
		line, err := scanner.ReadString('\n')
		if err == io.EOF {
			break
		}
		fields := strings.Fields(line)
		if len(fields) < 3 {
			return nil, fmt.Errorf("wrong number of fields (expected at least 3, got %d): %s", len(fields), line)
		}
		lineNum++
		if lineNum == 1 {
			continue // skip header
		}
		routes = append(routes, Route{})
		route := &routes[len(routes)-1]
		route.Interface = fields[0]
		ip, err := parseIP(fields[1])
		if err != nil {
			return nil, err
		}
		route.Destination = ip
		ip, err = parseIP(fields[2])
		if err != nil {
			return nil, err
		}
		route.Gateway = ip
	}
	return routes, nil
}

func getDefaultRoutes() map[string]net.IP {
	routes, err := GetRoutes()
	if err != nil {
		fmt.Printf("ERROR: %v", err)
		os.Exit(1)
	}

	defaultRoutes := make(map[string]net.IP)

	for i := range routes {
		zero := net.IP{0, 0, 0, 0}
		if routes[i].Destination.Equal(zero) {
			defaultRoutes[routes[i].Interface] = routes[i].Gateway
		}
	}

	return defaultRoutes
}

func localAddresses() ([]iface, error) {
	var interfaceList []iface

	ifaces, err := net.Interfaces()
	if err != nil {
		log.Print(fmt.Errorf("localAddresses: %v\n", err.Error()))
		return interfaceList, err
	}

	for _, i := range ifaces {
		// Skip interfaces that don't have a MAC address
		if i.HardwareAddr.String() == "" {
			continue
		}

		addrs, err := i.Addrs()
		if err != nil {
			log.Print(fmt.Errorf("localAddresses: %v\n", err.Error()))
			continue
		}

		for _, a := range addrs {
			i := iface{name: i.Name, mac: i.HardwareAddr.String(), addr: a.String()}
			interfaceList = append(interfaceList, i)
		}
	}

	return interfaceList, nil
}

func main() {
	defaultRoutes := getDefaultRoutes()

	ifaces, err := localAddresses()
	if err != nil {
		log.Printf("Error getting interfaces: %s", err.Error())
		os.Exit(1)
	}

	for _, i := range ifaces {
		ip, _, _ := net.ParseCIDR(i.addr)
		if ip.To4() == nil {
			log.Printf("Skipping non-IPv4 address: %s\n", i.addr)
			continue
		}

		gw := defaultRoutes[i.name]
		if gw == nil {
			log.Printf("Skipping IP because couldn't find default gateway for its interface: %s (iface: %s)\n", i.addr, i.name)
			continue
		}

		//                   IFACE   SOURCE     GATEWAY
		// arping -U -c 1 -I eth0 -s 69.162.98.2 69.162.98.1
		//
		// 2: eth0:
		//    link/ether 00:27:0e:09:7f:63 brd ff:ff:ff:ff:ff:ff
		//    inet 69.162.98.2/24 brd 69.162.98.255 scope global eth0
		//
		// Who has 69.162.98.1? Tell 69.162.98.2
		// - Sender MAC: 00:27:0e:09:7f:63 (eth0)  <- me
		// - Sender IP: 69.162.98.2                <- me
		// - Target MAC: ff:ff:ff:ff:ff:ff         <- everybody
		// - Target IP: 69.162.98.1                <- gateway
		//
		// Asking everybody who has the gateway's IP address causes everbody to see
		// who asked it and thus everybody learns that MAC/IP go together.
		log.Printf("Executing: arping -U -c 1 -I %s -s %s %s\n", i.name, ip, gw.String())

		args := []string{"-U", "-c", "1", "-I", i.name, "-s", ip.String(), gw.String()}
		output, err := exec.Command("arping", args...).Output()
		if err != nil {
			log.Printf("Error running command: %s", err.Error())
			os.Exit(1)
		}
		fmt.Println(string(output))
	}
}
