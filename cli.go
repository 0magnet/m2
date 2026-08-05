// Package main cli.go
package main

import (
	_ "embed"
	"fmt"
	"log"
	"os"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/0magnet/calvin"
	"github.com/bitfield/script"
	cc "github.com/ivanpirog/coloredcobra"
	"github.com/spf13/cobra"
	"github.com/stripe/stripe-go/v81"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

func init() {
	stripe.EnableTelemetry = false
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	rootCmd.AddCommand(
		runCmd,
		genCmd,
		wasmCmd,
	)
	var helpflag bool
	rootCmd.SetUsageTemplate(help)
	rootCmd.PersistentFlags().BoolVarP(&helpflag, "help", "h", false, "help for "+rootCmd.Use)
	rootCmd.SetHelpCommand(&cobra.Command{Hidden: true})
	rootCmd.PersistentFlags().MarkHidden("help") //nolint

}

var rootCmd = &cobra.Command{
	Use:   "m2",
	Short: "web store server",
	Long:  calvin.AsciiFont("magnetosphere.net") + "\n" + "web store server",
}

var genCmd = &cobra.Command{
	Use:   "gen",
	Short: "generate conf template",
	Long:  "generate conf template",
	Run: func(_ *cobra.Command, _ []string) {
		fmt.Println(envfiletemplate)
	},
}

// Execute executes the root cli command
func Execute() {
	cc.Init(&cc.Config{
		RootCmd:         rootCmd,
		Headings:        cc.HiBlue + cc.Bold,
		Commands:        cc.HiBlue + cc.Bold,
		CmdShortDescr:   cc.HiBlue,
		Example:         cc.HiBlue + cc.Italic,
		ExecName:        cc.HiBlue + cc.Bold,
		Flags:           cc.HiBlue + cc.Bold,
		FlagsDescr:      cc.HiBlue,
		NoExtraNewlines: true,
		NoBottomNewline: true,
	})
	if err := rootCmd.Execute(); err != nil {
		log.Fatal("Failed to execute command: ", err)
	}
}

var menvfile = os.Getenv("MENV")

type flagVars struct {
	Teststripekey      bool
	ProductsCSV        string
	WebPort            int
	CoreRunWebPort     int
	StripelivePK       string
	StripeliveSK       string
	StripetestPK       string
	StripetestSK       string
	StripeSK           string
	StripePK           string
	Siteimagesrc       string
	Siteordersurl      string
	Sitename           string
	Siteext            string
	Sitedomain         string
	Sitelongname       string
	Sitetagline        string
	Sitemeta           string
	Siteprettyname     string
	Siteprettynamecap  string
	Siteprettynamecaps string
	SiteASCIILogo      string
	Tgcontact          string
	Tgchannel          string
	UseTinygo          bool
	WasmSRC            []string
	WasmExecPath       string
	WasmExecPathGo     string
	WasmExecPathTinyGo string
	Gobuild            string
	Tinygobuild        string
	Buildwasmwith      string
	LDFlagsX           string
	NoCore             bool          // disable cogentcore UI build and serving
	PagesUI            bool          // use programmatic pages UI instead of content system
	PrinterName        string        // CUPS queue name (blank = default)
	CupsOptions        string        // comma-separated -o options
	LpTimeout          time.Duration // timeout for `lp`
}

var f = flagVars{
	//	WasmSRC: []string{"wasm/stl2.go","wasm/checkout_wasm.go"},
	WasmExecPath:       runtime.GOROOT() + "/lib/wasm/wasm_exec.js",                                    //nolint
	WasmExecPathGo:     runtime.GOROOT() + "/lib/wasm/wasm_exec.js",                                    //nolint
	WasmExecPathTinyGo: strings.TrimSuffix(runtime.GOROOT(), "go") + "tinygo" + "/targets/wasm_exec.js", //nolint
	Gobuild:            "go build",
	Tinygobuild:        "tinygo build -target=wasm --no-debug",
	Buildwasmwith:      "go build",
	LDFlagsX:           "stripePK",
}

var (
	// Hardcoded array of valid shorthand characters, excluding "h"
	shorthandChars = []rune("abcdefgijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	nextShortIndex = 0 // Index for the next shorthand flag
)

// Get the next available shorthand flag
func getNextShortFlag() string {
	if nextShortIndex >= len(shorthandChars) {
		return ""
	}
	short := shorthandChars[nextShortIndex]
	nextShortIndex++
	return string(short)
}

var a = true
var b = false

func addStringFlag(cmds []*cobra.Command, f interface{}, fieldPtr *string, description string) {
	for i, _ := range cmds {
		cmds[i].Flags().StringVarP(fieldPtr, ccc(fieldPtr, f, b), getNextShortFlag(), scriptExecString(fmt.Sprintf("${%s%s}", ccc(fieldPtr, f, a), func(s string) string {
			if s != "" {
				s = "-" + s
			}
			return s
		}(*fieldPtr))), fmt.Sprintf("%s env: %s\033[0m\n\r", description, ccc(fieldPtr, f, a)))
	}
}

func addStringSliceFlag(cmds []*cobra.Command, f interface{}, fieldPtr *[]string, description string) {
	for i, _ := range cmds {
		cmds[i].Flags().StringSliceVarP(
			fieldPtr,
			ccc(fieldPtr, f, b),
			getNextShortFlag(),
			scriptExecStringSlice(fmt.Sprintf("${%s[@]}", ccc(fieldPtr, f, a))),
			fmt.Sprintf("%s env: %s\033[0m\n\r", description, ccc(fieldPtr, f, a)),
		)
	}
}

func addBoolFlag(cmds []*cobra.Command, f interface{}, fieldPtr *bool, description string) {
	for i, _ := range cmds {
		cmds[i].Flags().BoolVarP(fieldPtr, ccc(fieldPtr, f, b), getNextShortFlag(), scriptExecBool(fmt.Sprintf("${%s%s}", ccc(fieldPtr, f, a), func(b bool) string {
			return "-" + strconv.FormatBool(b)
		}(*fieldPtr))), fmt.Sprintf("%s env: %s\033[0m\n\r", description, ccc(fieldPtr, f, a)))
	}
}
func addIntFlag(cmds []*cobra.Command, f interface{}, fieldPtr *int, description string) {
	for i, _ := range cmds {
		cmds[i].Flags().IntVarP(fieldPtr, ccc(fieldPtr, f, b), getNextShortFlag(), scriptExecInt(fmt.Sprintf("${%s%s}", ccc(fieldPtr, f, a), func(i int) string {
			return fmt.Sprintf("-%d", i)
		}(*fieldPtr))), fmt.Sprintf("%s env: %s\033[0m\n\r", description, ccc(fieldPtr, f, a)))
	}
}

func addDurationFlag(cmds []*cobra.Command, f interface{}, fieldPtr *time.Duration, description string) {
	for i := range cmds {
		// Keep parity with your pattern of embedding a "-" when a non-zero default is present.
		def := scriptExecDuration(fmt.Sprintf("${%s%s}",
			ccc(fieldPtr, f, a),
			func(d time.Duration) string {
				if d != 0 {
					return "-" + d.String() // e.g. "-5s"
				}
				return ""
			}(*fieldPtr),
		))
		cmds[i].Flags().DurationVarP(
			fieldPtr,
			ccc(fieldPtr, f, b),
			getNextShortFlag(),
			def,
			fmt.Sprintf("%s env: %s\033[0m\n\r", description, ccc(fieldPtr, f, a)),
		)
	}
}

func init() {
	runCmd.Flags().SortFlags = false
	addStringFlag([]*cobra.Command{runCmd}, &f, &f.ProductsCSV, "products csv file")
	addBoolFlag([]*cobra.Command{runCmd, wasmCmd}, &f, &f.Teststripekey, "use stripe test api keys instead of live key")
	addStringFlag([]*cobra.Command{runCmd, wasmCmd}, &f, &f.StripeliveSK, "stripe live api sk")
	addStringFlag([]*cobra.Command{runCmd, wasmCmd}, &f, &f.StripelivePK, "stripe live api pk")
	addStringFlag([]*cobra.Command{runCmd, wasmCmd}, &f, &f.StripetestSK, "stripe test api sk")
	addStringFlag([]*cobra.Command{runCmd, wasmCmd}, &f, &f.StripetestPK, "stripe test api pk")
	addIntFlag([]*cobra.Command{runCmd}, &f, &f.WebPort, "port to serve on")
	addStringFlag([]*cobra.Command{runCmd}, &f, &f.Siteimagesrc, "domain for images - leave blank to serve images")
	addStringFlag([]*cobra.Command{runCmd}, &f, &f.Siteordersurl, "domain for orders - leave blank for same domain")
	addStringFlag([]*cobra.Command{runCmd}, &f, &f.Sitename, "site name")
	addStringFlag([]*cobra.Command{runCmd}, &f, &f.Siteext, "site domain extension")
	addStringFlag([]*cobra.Command{runCmd}, &f, &f.Sitelongname, "site long name")
	addStringFlag([]*cobra.Command{runCmd}, &f, &f.Sitetagline, "site tag line")
	addStringFlag([]*cobra.Command{runCmd}, &f, &f.Sitemeta, "site meta")
	addStringFlag([]*cobra.Command{runCmd}, &f, &f.Tgcontact, "telegram contact")
	addStringFlag([]*cobra.Command{runCmd}, &f, &f.Tgchannel, "telegram channel")
	addBoolFlag([]*cobra.Command{runCmd}, &f, &f.NoCore, "disable cogentcore UI build and serving")
	addBoolFlag([]*cobra.Command{runCmd}, &f, &f.PagesUI, "use programmatic pages UI instead of content system")
	addBoolFlag([]*cobra.Command{runCmd}, &f, &f.UseTinygo, "use tinygo instead of go to compile wasm")
	addStringSliceFlag([]*cobra.Command{runCmd, wasmCmd}, &f, &f.WasmSRC, "wasm source code files RELATIVE PATHS without '..'")
	addStringFlag([]*cobra.Command{runCmd}, &f, &f.PrinterName, "CUPS printer name (default: system default)")
	addStringFlag([]*cobra.Command{runCmd}, &f, &f.CupsOptions, "e.g. 'media=Custom.80x200mm,fit-to-page'")
	addDurationFlag([]*cobra.Command{runCmd}, &f, &f.LpTimeout, "timeout for lp command")

}

// change case
func ccc(val interface{}, strct interface{}, upper bool) string {
	v := reflect.ValueOf(strct)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		panic("uc: second argument must be a pointer to a struct")
	}
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		if field.CanAddr() && field.Addr().Interface() == val {
			if upper {
				return strings.ToUpper(v.Type().Field(i).Name)
			}
			return strings.ToLower(v.Type().Field(i).Name)
		}
	}
	return ""
}

func initstripePK() {
	f.StripeSK = f.StripeliveSK
	f.StripePK = f.StripelivePK
	if f.Teststripekey {
		f.StripeSK = f.StripetestSK
		f.StripePK = f.StripetestPK
	}
	stripe.Key = f.StripeSK
	// awkward way to do this
	f.LDFlagsX += "=" + f.StripePK
}

var wasmCmd = &cobra.Command{
	Use:   "wasm",
	Short: "compile wasm",
	Long:  "compile wasm",
	Run: func(_ *cobra.Command, _ []string) {
		if len(f.WasmSRC) == 0 {
			log.Fatal("No wasm source code specified")
		}
		initstripePK()
		compileWASM()
	},
}

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "run the web application",
	Long: calvin.AsciiFont("magnetosphere.net") + "\n" + func() string {
		helptext := `Run the web application
Generate a config file first

Config defaults file may also be specified with:
MENV=m2.conf m2 run
OR
MENV=/path/to/m2.conf m2 run
print the MENV file template with:
m2 gen`
		if menvfile == "" {
			return helptext
		}
		if _, err := os.Stat(menvfile); err == nil {
			return `Run the web application

menv file detected: ` + menvfile
		}
		return helptext
	}(),
	Run: func(_ *cobra.Command, _ []string) {
		f.Sitedomain = f.Sitename + f.Siteext
		log.Println(" Initializing " + f.Sitedomain)
		fmt.Println(calvin.BlackboardBold(f.Sitedomain))
		fmt.Println(calvin.AsciiFont(f.Sitedomain))
		initstripePK()
		f.Siteprettyname = calvin.BlackboardBold(f.Sitedomain) //"𝕄𝕒𝕘𝕟𝕖𝕥𝕠𝕤𝕡𝕙𝕖𝕣𝕖.𝕟𝕖𝕥"
		c := cases.Title(language.English)
		f.Siteprettynamecap = calvin.BlackboardBold(c.String(f.Sitedomain))         //"𝕄𝕒𝕘𝕟𝕖𝕥𝕠𝕤𝕡𝕙𝕖𝕣𝕖.𝕟𝕖𝕥"
		f.Siteprettynamecaps = calvin.BlackboardBold(strings.ToUpper(f.Sitedomain)) //"𝕄𝔸𝔾ℕ𝔼𝕋𝕆𝕊ℙℍ𝔼ℝ𝔼.ℕ𝔼𝕋"
		f.SiteASCIILogo = strings.Replace(strings.Replace(calvin.AsciiFont(f.Sitedomain), " ", "&nbsp;", -1), "\n", "<br>\n", -1)

		if f.UseTinygo {
			f.WasmExecPath = f.WasmExecPathTinyGo
			f.Buildwasmwith = f.Tinygobuild
		}
		if len(f.WasmSRC) == 0 {
			f.WasmExecPath = ""
			f.Buildwasmwith = ""
		}
		log.Println("Checking for products CSV")
		fileInfo, err := os.Stat(f.ProductsCSV)
		if err != nil {
			log.Fatal("Error getting file info:", err)
		}
		lastModTime = fileInfo.ModTime()
		log.Println("Reading products CSV")
		prods := readCSV(f.ProductsCSV)
		if warnings := validateCSV(prods); len(warnings) > 0 {
			for _, w := range warnings {
				log.Println("CSV warning:", w)
			}
		}
		allproductsMu.Lock()
		allproducts = prods
		allproductsMu.Unlock()
		go func() {
			for {
				fileInfo, err := os.Stat(f.ProductsCSV)
				if err != nil {
					log.Println("Error getting file info:", err)
					time.Sleep(10 * time.Second)
					continue
				}

				currentModTime := fileInfo.ModTime()
				if currentModTime != lastModTime {
					log.Println("CSV file has been modified!")
					prods := readCSV(f.ProductsCSV)
					if warnings := validateCSV(prods); len(warnings) > 0 {
						for _, w := range warnings {
							log.Println("CSV warning:", w)
						}
					}
					allproductsMu.Lock()
					allproducts = prods
					allproductsMu.Unlock()
					lastModTime = currentModTime
					if !f.NoCore {
						createCORE()
						buildCORE()
					}
				}

				time.Sleep(10 * time.Second)
			}
		}()

		server()
	},
}

var lastModTime time.Time

func scriptExecString(s string) string {
	z, err := script.Exec(fmt.Sprintf(`bash -c 'MENV=%s ; if [[ $MENV != "" ]] && [[ -f $MENV ]] ; then source $MENV ; fi ; printf "%s"'`, menvfile, s)).String()
	if err == nil {
		return strings.TrimSpace(z)
	}
	return ""
}

func scriptExecStringSlice(s string) []string {
	z, err := script.Exec(fmt.Sprintf(`bash -c 'MENV=%s ; if [[ $MENV != "" ]] && [[ -f $MENV ]] ; then source $MENV ; fi ; printf "%s" "%s"'`, menvfile, "%s\n", s)).Slice()
	if err == nil {
		return z
	}
	return []string{""}
}

func scriptExecBool(s string) bool {
	z, err := script.Exec(fmt.Sprintf(`bash -c 'MENV=%s ; if [[ $MENV != "" ]] && [[ -f $MENV ]] ; then source $MENV ; fi ; printf "%s"'`, menvfile, s)).String()
	if err == nil {
		b, err := strconv.ParseBool(z)
		if err == nil {
			return b
		}
	}
	return false
}

/*
func scriptExecArray(s string) string {
	y, err := script.Exec(fmt.Sprintf(`bash -c 'MENV=%s ; if [[ $MENV != "" ]] && [[ -f $MENV ]] ; then source $MENV ; fi ; for _i in %s ; do echo "$_i" ; done'`, menvfile, s)).Slice()
	if err == nil {
		return strings.Join(y, ",")
	}
	return ""
}
*/

func scriptExecInt(s string) int {
	z, err := script.Exec(fmt.Sprintf(`bash -c 'MENV=%s ; if [[ $MENV != "" ]] && [[ -f $MENV ]] ; then source $MENV ; fi ; printf "%s"'`, menvfile, s)).String()
	if err == nil {
		if z == "" {
			return 0
		}
		i, err := strconv.Atoi(z)
		if err == nil {
			return i
		}
	}
	return 0
}

// Accepts Go duration strings ("750ms", "2s", "5m", "1h").
// Also accepts a bare integer (treated as seconds).
// If the evaluated string starts with "-", it is trimmed (matching your other helpers’ default building).
func scriptExecDuration(s string) time.Duration {
	z, err := script.Exec(fmt.Sprintf(`bash -c 'MENV=%s ; if [[ $MENV != "" ]] && [[ -f $MENV ]] ; then source $MENV ; fi ; printf "%s"'`, menvfile, s)).String()
	if err != nil {
		return 0
	}
	z = strings.TrimSpace(z)
	if z == "" {
		return 0
	}
	z = strings.TrimPrefix(z, "-") // keep parity with how defaults are built

	// Try full Go duration syntax first.
	if d, err := time.ParseDuration(z); err == nil {
		return d
	}
	// Fallback: plain integer means seconds.
	if n, err := strconv.ParseInt(z, 10, 64); err == nil {
		return time.Duration(n) * time.Second
	}
	return 0
}

const help = "\r\n" +
	"  {{if .HasAvailableSubCommands}}{{end}} {{if gt (len .Aliases) 0}}\r\n\r\n" +
	"{{.NameAndAliases}}{{end}}{{if .HasAvailableSubCommands}}\r\n\r\n" +
	"Available Commands:{{range .Commands}}{{if (or .IsAvailableCommand)}}\r\n  " +
	"{{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}\r\n\r\n" +
	"Flags:\r\n" +
	"{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}\r\n\r\n" +
	"Global Flags:\r\n" +
	"{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}\r\n\r\n"

const envfiletemplate = `#########################################################################
#	M2 CONFIG
#
# Copy to <yoursite>.conf and run with:  MENV=<yoursite>.conf m2 run
# This file is sourced by bash; use shell syntax.
# Comment a value with # to use the built-in default.
#########################################################################

### Stripe Configuration ################################################

#-- Live and test API keys - REQUIRED for checkout
# https://dashboard.stripe.com/apikeys
STRIPELIVEPK='pk_live_...'
STRIPELIVESK='sk_live_...'
STRIPETESTPK='pk_test_...'
STRIPETESTSK='sk_test_...'

#-- Use the test keys instead of the live keys
TESTSTRIPEKEY=true

### Product Data ########################################################

#-- Products CSV path (see products.example.csv for the schema)
PRODUCTSCSV='products.csv'

### Site Identity #######################################################

#-- Image subdomain, no trailing slash (ex. 'https://img.example.com')
# empty = serve images from ./img
SITEIMAGESRC=''

#-- Orders subdomain, no trailing slash (ex. 'https://pay.example.com')
SITEORDERSURL=''

#-- Website (Host) Name - domain minus extension (ex. 'example')
SITENAME='example'

#-- Website Domain Extension (ex. '.com' '.net')
SITEEXT='.com'

#-- Site Long Name (ex. 'example electronic surplus')
SITELONGNAME='example web store'

#-- Site Tag Line
SITETAGLINE='an example web store'

#-- Site Meta Description (SEO)
SITEMETA='an example web store selling example things'

#-- Telegram contact + channel; username only, no 'https://t.me/'
TGCONTACT=''
TGCHANNEL=''

### Web Server ##########################################################

#-- Port to serve http on
WEBPORT='9883'

#-- Port for the cogentcore UI dev server
CORERUNWEBPORT='8536'

#-- Disable the cogentcore UI build + serving; hides the Interface link
NOCORE=true

#-- Use programmatic pages UI instead of content system
PAGESUI=false

### WebAssembly #########################################################

#-- Compile wasm with tinygo (smaller output) in addition to go
USETINYGO=true

#-- wasm source directories, relative paths
# 'wasm/cart' powers checkout — keep it. Additional entries are drop-in
# wasm apps: any dir under wasm/ with a main package (a self-contained
# module with its own vendor/ also works). Empty () disables all wasm.
WASMSRC=('wasm/cart')

### Receipt Printing (CUPS) #############################################

#-- CUPS printer name (default: system default)
PRINTERNAME=''

#-- CUPS options, comma separated (ex. 'media=Custom.80x200mm,fit-to-page')
CUPSOPTIONS=''

#-- timeout for lp command
LPTIMEOUT='10s'
`
