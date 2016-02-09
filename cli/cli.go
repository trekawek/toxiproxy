package main

import (
	"sort"
	"strconv"
	"strings"

	"github.com/Shopify/toxiproxy/client"
	"github.com/codegangsta/cli"

	"fmt"
	"os"
)

const (
	redColor    = "\x1b[31m"
	greenColor  = "\x1b[32m"
	yellowColor = "\x1b[33m"
	blueColor   = "\x1b[34m"
	cyanColor   = "\x1b[36m"
	purpleColor = "\x1b[35m"
	grayColor   = "\x1b[37m"
	noColor     = "\x1b[0m"
)

var ToxicDescription = `
	Default Toxics:
	latency:	delay all data +/- jitter
		latency=<ms>,jitter=<ms>

	bandwidth:	limit to max kb/s
		rate=<KB/s>

	slow_close:	delay from closing
		delay=<ms>

	timeout: 	stop all data and close after timeout
	         	timeout=<ms>

	slicer: 	slice data into bits with optional delay
	        	average_size=<byes>,size_variation=<bytes>,delay=<microseconds>

	toxic add:
		usage: toxiproxy-client add <proxyName> --type <toxicType> --toxicName <toxicName> \
		--fields <key1=value1,key2=value2...> --upstream --downstream

		example: toxiproxy-client toxic add myProxy -t latency -n myToxic -f latency=100,jitter=50

	toxic update:
		usage: toxiproxy-client update <proxyName> --toxicName <toxicName> \
		--fields <key1=value1,key2=value2...>
		
		example: toxiproxy-client toxic update myProxy -n myToxic -f jitter=25
	
	toxic delete:
		usage: toxiproxy-client update <proxyName> --toxicName <toxicName>

		example: toxiproxy-client toxic delete myProxy -n myToxic
`

func main() {
	toxiproxyClient := toxiproxy.NewClient("http://localhost:8474")

	app := cli.NewApp()
	app.Name = "Toxiproxy"
	app.Version = "2.0"
	app.Usage = "Simulate network and system conditions"
	app.Commands = []cli.Command{
		{
			Name:    "list",
			Usage:   "list all proxies\n\tusage: 'toxiproxy-client list'\n",
			Aliases: []string{"l", "li", "ls"},
			Action:  withToxi(list, toxiproxyClient),
		},
		{
			Name:    "inspect",
			Aliases: []string{"i", "ins"},
			Usage:   "inspect a single proxy\n\tusage: 'toxiproxy-client inspect <proxyName>'\n",
			Action:  withToxi(inspect, toxiproxyClient),
		},
		{
			Name:    "toggle",
			Usage:   "\ttoggle enabled status on a proxy\n\t\tusage: 'toxiproxy-client toggle <proxyName>'\n",
			Aliases: []string{"tog"},
			Action:  withToxi(toggle, toxiproxyClient),
		},
		{
			Name:    "create",
			Usage:   "create a new proxy\n\tusage: 'toxiproxy-client create <proxyName> --listen <addr> --downstream <addr>'\n",
			Aliases: []string{"c", "new"},
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "listen, l",
					Usage: "proxy will listen on this address",
				},
				cli.StringFlag{
					Name:  "upstream, u",
					Usage: "proxy will forward to this address",
				},
			},
			Action: withToxi(create, toxiproxyClient),
		},
		{
			Name:    "delete",
			Usage:   "\tdelete a proxy\n\t\tusage: 'toxiproxy-client delete <proxyName>'\n",
			Aliases: []string{"d"},
			Action:  withToxi(delete, toxiproxyClient),
		},
		{
			Name:        "toxic",
			Aliases:     []string{"t"},
			Usage:       "\tadd, remove or update a toxic\n\t\tusage: see 'toxiproxy-client toxic'\n",
			Description: ToxicDescription,
			Subcommands: []cli.Command{
				{
					Name:    "add",
					Aliases: []string{"a"},
					Usage:   "add a new toxic",
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:  "toxicName, n",
							Usage: "name of the toxic",
						},
						cli.StringFlag{
							Name:  "type, t",
							Usage: "type of toxic",
						},
						cli.StringFlag{
							Name:  "fields, f",
							Usage: "comma seperated key=value toxic fields",
						},
						cli.BoolFlag{
							Name:  "upstream, u",
							Usage: "add toxic to upstream",
						},
						cli.BoolFlag{
							Name:  "downstream, d",
							Usage: "add toxic to downstream",
						},
					},
					Action: withToxi(addToxic, toxiproxyClient),
				},
				{
					Name:    "update",
					Aliases: []string{"u"},
					Usage:   "update an enabled toxic",
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:  "toxicName, n",
							Usage: "name of the toxic",
						},
						cli.StringFlag{
							Name:  "fields, f",
							Usage: "comma seperated key=value toxic fields",
						},
					},
					Action: withToxi(updateToxic, toxiproxyClient),
				},
				{
					Name:    "remove",
					Aliases: []string{"r", "delete", "d"},
					Usage:   "remove an enabled toxic",
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:  "toxicName, n",
							Usage: "name of the toxic",
						},
					},
					Action: withToxi(removeToxic, toxiproxyClient),
				},
			},
		},
	}

	app.Run(os.Args)
}

type toxiAction func(*cli.Context, *toxiproxy.Client)

func withToxi(f toxiAction, t *toxiproxy.Client) func(*cli.Context) {
	return func(c *cli.Context) {
		f(c, t)
	}
}

func list(c *cli.Context, t *toxiproxy.Client) {
	proxies, err := t.Proxies()
	if err != nil {
		fatalf("Failed to retrieve proxies: %s", err)
	}

	var proxyNames []string
	for proxyName := range proxies {
		proxyNames = append(proxyNames, proxyName)
	}
	sort.Strings(proxyNames)

	fmt.Fprintf(os.Stderr, "%sListen\t\t%sUpstream\t%sName\t%sEnabled\t%sToxics\n%s", blueColor, yellowColor,
		greenColor, purpleColor, redColor, noColor)
	fmt.Fprintf(os.Stderr, "%s================================================================================%s\n", noColor, noColor)

	if len(proxyNames) == 0 {
		fmt.Printf("%sno proxies\n\n%s", redColor, noColor)
		hint("create a proxy with `toxiproxy-client create`")
		return
	}

	for _, proxyName := range proxyNames {
		proxy := proxies[proxyName]
		numToxics := strconv.Itoa(len(proxy.ActiveToxics))
		if numToxics == "0" {
			numToxics = "None"
		}
		fmt.Printf("%s%s\t%s%s\t%s%s\t%s%v\t%s%s%s\n", blueColor, proxy.Listen, yellowColor, proxy.Upstream,
			enabledColor(proxy.Enabled), proxy.Name, purpleColor, proxy.Enabled, redColor, numToxics, noColor)
	}
	fmt.Println()
	hint("inspect a toxic with `toxiproxy-client inspect <proxyName>`")
}
func inspect(c *cli.Context, t *toxiproxy.Client) {
	proxyName := c.Args().First()
	if proxyName == "" {
		fatalf("Proxy name is required as the first argument.\n")
	}

	proxy, err := t.Proxy(proxyName)
	if err != nil {
		fatalf("Failed to retrieve proxy %s: %s\n", proxyName, err.Error())
	}

	fmt.Printf("%sproxy name: %s%s%s\n", noColor, enabledColor(proxy.Enabled), proxy.Name, noColor)
	fmt.Printf("%slisten: %s%s %s---> upstream: %s%s%s\n\n", noColor, blueColor, proxy.Listen, noColor, yellowColor, proxy.Upstream, noColor)

	splitToxics := func(toxics toxiproxy.Toxics) (toxiproxy.Toxics, toxiproxy.Toxics) {
		upstream := make(toxiproxy.Toxics)
		downstream := make(toxiproxy.Toxics)
		for k, toxic := range toxics {
			if toxic["stream"].(string) == "upstream" {
				upstream[k] = toxic
			} else {
				downstream[k] = toxic
			}
		}
		return upstream, downstream
	}

	if len(proxy.ActiveToxics) == 0 {
		fmt.Printf("%sno toxics\n%s", redColor, noColor)
	} else {
		up, down := splitToxics(proxy.ActiveToxics)
		listToxics(up, "upstream")
		fmt.Println()
		listToxics(down, "downstream")
	}
	fmt.Println()

	hint("add a toxic with `toxiproxy-client toxic add`")
}

func toggle(c *cli.Context, t *toxiproxy.Client) {
	proxyName := c.Args().First()
	if proxyName == "" {
		fatalf("Proxy name is required as the first argument.\n")
	}

	proxy, err := t.Proxy(proxyName)
	if err != nil {
		fatalf("Failed to retrieve proxy %s: %s\n", proxyName, err.Error())
	}

	proxy.Enabled = !proxy.Enabled

	err = proxy.Save()
	if err != nil {
		fatalf("Failed to toggle proxy %s: %s\n", proxyName, err.Error())
	}

	fmt.Printf("Proxy %s%s%s is now %s%s%s\n", enabledColor(proxy.Enabled), proxyName, noColor, enabledColor(proxy.Enabled), enabledText(proxy.Enabled), noColor)
}

func create(c *cli.Context, t *toxiproxy.Client) {
	proxyName := c.Args().First()
	if proxyName == "" {
		fatalf("Proxy name is required as the first argument.\n")
	}
	listen := getArgOrFail(c, "listen")
	upstream := getArgOrFail(c, "upstream")
	_, err := t.CreateProxy(proxyName, listen, upstream)
	if err != nil {
		fatalf("Failed to create proxy: %s\n", err.Error())
	}
	fmt.Printf("Created new proxy %s\n", proxyName)
}

func delete(c *cli.Context, t *toxiproxy.Client) {
	proxyName := c.Args().First()
	if proxyName == "" {
		fatalf("Proxy name is required as the first argument.\n")
	}
	p, err := t.Proxy(proxyName)
	if err != nil {
		fatalf("Failed to retrieve proxy %s: %s\n", proxyName, err.Error())
	}

	err = p.Delete()
	if err != nil {
		fatalf("Failed to delete proxy: %s\n", err.Error())
	}
	fmt.Printf("Deleted proxy %s\n", proxyName)
}

func addToxic(c *cli.Context, t *toxiproxy.Client) {
	proxyName := c.Args().First()
	if proxyName == "" {
		fatalf("Proxy name is required as the first argument.\n")
	}
	toxicName := c.String("toxicName")
	toxicType := getArgOrFail(c, "type")
	toxicFields := getArgOrFail(c, "fields")

	upstream := c.Bool("upstream")
	downstream := c.Bool("downstream")

	fields := parseFields(toxicFields)

	p, err := t.Proxy(proxyName)
	if err != nil {
		fatalf("Failed to retrieve proxy %s: %s\n", proxyName, err.Error())
	}

	addToxic := func(stream string) {
		t, err := p.AddToxic(toxicName, toxicType, stream, fields)
		if err != nil {
			fatalf("Failed to add toxic: %s\n", err.Error())
		}
		toxicName = t["name"].(string)
		fmt.Printf("Added %s %s toxic '%s' on proxy '%s'\n", stream, toxicType, toxicName, proxyName)
	}

	if upstream {
		addToxic("upstream")
	}
	// Default to downstream.
	if downstream || (!downstream && !upstream) {
		addToxic("downstream")
	}
}

func updateToxic(c *cli.Context, t *toxiproxy.Client) {
	proxyName := c.Args().First()
	if proxyName == "" {
		fatalf("Proxy name is required as the first argument.\n")
	}
	toxicName := getArgOrFail(c, "toxicName")
	toxicFields := getArgOrFail(c, "fields")

	fields := parseFields(toxicFields)

	p, err := t.Proxy(proxyName)
	if err != nil {
		fatalf("Failed to retrieve proxy %s: %s\n", proxyName, err.Error())
	}

	_, err = p.UpdateToxic(toxicName, fields)
	if err != nil {
		fatalf("Failed to update toxic: %s\n", err.Error())
	}

	fmt.Printf("Updated toxic '%s' on proxy '%s'\n", toxicName, proxyName)
}

func removeToxic(c *cli.Context, t *toxiproxy.Client) {
	proxyName := c.Args().First()
	if proxyName == "" {
		fatalf("Proxy name is required as the first argument.\n")
	}
	toxicName := getArgOrFail(c, "toxicName")

	p, err := t.Proxy(proxyName)
	if err != nil {
		fatalf("Failed to retrieve proxy %s: %s\n", proxyName, err.Error())
	}

	err = p.RemoveToxic(toxicName)
	if err != nil {
		fatalf("Failed to remove toxic: %s\n", err.Error())
	}

	fmt.Printf("Removed toxic '%s' on proxy '%s'\n", toxicName, proxyName)
}

func parseFields(raw string) toxiproxy.Toxic {
	parsed := map[string]interface{}{}
	keyValues := strings.Split(raw, ",")
	for _, keyValue := range keyValues {
		kv := strings.SplitN(keyValue, "=", 2)
		if len(kv) != 2 {
			fatalf("Fields should be in format key1=value1,key2=value2\n")
		}
		value, err := strconv.Atoi(kv[1])
		if err != nil {
			fatalf("Toxic field was expected to be an integer.\n")
		}
		parsed[kv[0]] = value
	}
	return parsed
}

func enabledColor(enabled bool) string {
	if enabled {
		return greenColor
	}

	return redColor
}

func enabledText(enabled bool) string {
	if enabled {
		return "enabled"
	}

	return "disabled"
}

func listToxics(toxics toxiproxy.Toxics, stream string) {
	if len(toxics) == 0 {
		fmt.Printf("%sno %s toxics\n%s", redColor, stream, noColor)
		return
	}
	fmt.Printf("%s%s toxics:\n", noColor, stream)
	for name, toxic := range toxics {
		if toxic["stream"].(string) != stream {
			continue
		}
		fmt.Printf("%s%s:%s", greenColor, name, noColor)
		for property, value := range toxic {
			fmt.Printf(" %s=", property)
			fmt.Print(value)
		}
		fmt.Println()
	}
}

func getArgOrFail(c *cli.Context, name string) string {
	arg := c.String(name)
	if arg == "" {
		fatalf("Required argument '%s' was empty.\n", name)
	}
	return arg
}

func hint(m string) {
	fmt.Printf("%sHint: %s\n%s", cyanColor, m, noColor)
}

func fatalf(m string, args ...interface{}) {
	fmt.Printf(m, args...)
	os.Exit(1)
}
