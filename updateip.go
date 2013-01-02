package main

import ("fmt"
	"net"
	"net/http"
	"io/ioutil"
	"log"
	"regexp"
	"bytes"
	"encoding/xml"
	"strings"
	"time"
	"encoding/json"
	)	

// Simple XML-RPC method
func xmlrpc(url string, methodName string, args ...string) string {
	// Marshal XML
	type Param struct {
		Entry string `xml:"value>string"`
	}
	type methodCall struct {
		XMLName   xml.Name `xml:"methodCall"`
		MethodName string  `xml:"methodName"`
		Params []Param `xml:"params>param"`

	}
	v := &methodCall{MethodName: methodName}
	v.Params = []Param{}
	for _, arg := range args {
		v.Params = append(v.Params, Param{arg})
	}
	b, _ := xml.Marshal(v)
	bs := bytes.NewBuffer([]byte(b))

	// Deliver Payload
	r, e := http.Post(url, "text/xml", bs)
	if e != nil {
		fmt.Println(e)
		return ""
	}
	defer r.Body.Close()

	// Get text of response
	body, e := ioutil.ReadAll(r.Body)
	if e != nil {
		fmt.Println(e)
		return ""
	}
	return string(body)

}

//checkIP checks dyndns.org and parses the HTML for IP address
func checkIP() string {
	// Get the HTML
	resp, err := http.Get("http://checkip.dyndns.org")
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	html := string(body)

	// Find the IP
	re := regexp.MustCompile(": ([^<]*)") //Current IP Address: 12.23.34.45</body>
	ip := re.FindStringSubmatch(html)[1]
	return ip
}

//wfLogin returns the session authentication token that is required to use the Webfaction API
func wfLogin(user, pass string) string {
	resp := xmlrpc("https://api.webfaction.com/", "login", user, pass)
	re := regexp.MustCompile("[a-z0-9]{32}")
	token := re.FindString(resp)
	return token

}

//wfCreateDNSOverride adds an entry to the domain
func wfCreateDNSOverride(token, domain, ip string) bool {
	resp := xmlrpc("https://api.webfaction.com/", "create_dns_override", token, domain, ip)
	if strings.Contains(resp, "faultCode") {
		fmt.Printf("faultCode exists, bad entry")
		return false
	}
	//fmt.Println(resp)

	return true
}

//wfDeleteDNSOverride deletes an entry from the domain
func wfDeleteDNSOverride(token, domain string) bool {
	resp := xmlrpc("https://api.webfaction.com/", "delete_dns_override", token, domain)
	if strings.Contains(resp, "faultCode") {
		fmt.Printf("faultCode exists, bad entry")
		return false
	}
	//fmt.Println(resp)

	return true
}

//wfUpdateDNSOverride clears the overrides and creates a new one
func wfUpdateDNSOverride(token, domain, ip string) bool {
	success := wfDeleteDNSOverride(token, domain)
	if !success {
		return false
	}
	success = wfCreateDNSOverride(token, domain, ip)
	return success
}

// Json configuration
//{"user":"username","pass":"password","domain":"mydomain.example.com"}
type JsonCfg struct{
	User, Pass, Domain string
}

func parseCfg(file []byte) (JsonCfg) {
	var t JsonCfg
	json.Unmarshal(file, &t) 
	//fmt.Println(t)
	return t
}

// main
func main() {
	start := time.Now()

	// Read and parse json configuration file
	file, e := ioutil.ReadFile("./updateip.json")
	if e != nil {
		fmt.Printf("File error: %v\n", e)
		return
	}
	config := parseCfg(file)
	user := config.User
	pass := config.Pass
	domain := config.Domain

	cip := make(chan string)
	cdip := make(chan string)
	ctoken := make(chan string)

	// Get the IP from Dyndns
	go func() {
		fmt.Printf("Checking Local IP...\n")
		ip := checkIP()
		fmt.Printf("Local IP is %s\n", ip)
		cip <- ip
	}()

	// Compare the IP to hostname
	go func() {
		fmt.Printf("Checking <%s> IP...\n", domain)
		checkDomainIp, err := net.LookupIP(domain)
		if err != nil {
			log.Fatal(err)
		}
		domainIp := checkDomainIp[0].String()
		fmt.Printf("Domain IP is %s\n", domainIp)
		cdip <- domainIp
	}()

	// Log into Webfaction
	go func(){
		fmt.Printf("Logging into Webfaction for <%s>...\n", user)
		token := wfLogin(user, pass)
		fmt.Printf("Login token is %s\n", token)
		ctoken <- token
	}()

	ip := <- cip
	domainIp := <- cdip
	if ip == domainIp {
		fmt.Println("IP unchanged.")	
	} else {

		token := <- ctoken
		// Update DNS for Webfaction
		fmt.Printf("Updating DNS Override for <%s>...", domain)
		success := wfUpdateDNSOverride(token, domain, ip)
		if success {
			fmt.Println("Done!")
		} else {
			fmt.Println("Failed.")
		}
	}
	fmt.Printf("%.2fs total\n", time.Since(start).Seconds())
}
