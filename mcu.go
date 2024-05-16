package main

/*
MCU command line interface and flash assistant

This program started as a wrapper for the following workflow:
__________________________________________________________________________________________________________________________________________________________________________________________
#!/bin/bash
udisksctl mount -b /dev/sdd1
tinygo flash -target=pico -ldflags "-X main.timeStamp='$(date '+%Y-%m-%dT%H:%M:%SZ')' -X main.multiDisplay='true' -X main.rtcFuture='false' -X main.offSet='9'" main.go && \
 sleep 3 && \
 sudo chmod a+rw $(echo /dev/ttyACM?) && \
 (go run mcu.go  -m $(echo /dev/ttyACM?) || tinygo monitor)
 _________________________________________________________________________________________________________________________________________________________________________________________

Help menu --flags for subcommands are auto-generated from the .go file referenced by the GOPROG env

*/

import (
	"bufio"
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"log"
	"os"
	"strings"
	"path/filepath"
	"sync"
	"time"

	"github.com/bitfield/script"
	cc "github.com/ivanpirog/coloredcobra"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/tarm/serial"
)

func main() {
	cc.Init(&cc.Config{
		RootCmd:         RootCmd,
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
	if err := RootCmd.Execute(); err != nil {
		log.Fatal("Failed to execute command: ", err)
	}
}

var (
	fset *token.FileSet
	node *ast.File
	goProg    string
	ttyUSB    string
	baud      int
	sleepTime time.Duration
	target    string
	blkDev    string
)

func init() {
	RootCmd.CompletionOptions.DisableDefaultCmd = true
	RootCmd.Flags().StringVarP(&ttyUSB, "ser", "m", "", "block device for serial interface (i.e. \"/dev/ttyACM0\")\nif unspecified serial connection will not be attempted")
	RootCmd.Flags().IntVarP(&baud, "baud", "b", 9600, "baud rate")
	var helpflag bool
	RootCmd.SetUsageTemplate(help)
	RootCmd.PersistentFlags().BoolVarP(&helpflag, "help", "h", false, "help for "+RootCmd.Use)
	RootCmd.SetHelpCommand(&cobra.Command{Hidden: true})
	RootCmd.PersistentFlags().MarkHidden("help") //nolint
}

// RootCmd is the root command
var RootCmd = &cobra.Command{
	Use: func() string {
		return strings.Split(filepath.Base(strings.ReplaceAll(strings.ReplaceAll(fmt.Sprintf("%v", os.Args), "[", ""), "]", "")), " ")[0]
	}(),
	Short:                 "mcu serial interfacer",
	Long:                  "mcu serial interfacer\n",
	SilenceErrors:         true,
	SilenceUsage:          true,
	DisableSuggestions:    true,
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		if ttyUSB == "" {
			cmd.Help()
			return
		}
		port, err := serial.OpenPort(&serial.Config{Name: ttyUSB, Baud: baud})
		if err != nil {
			log.Fatalf("serial.OpenPort: %v", err)
		}
		defer port.Close()
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			reader := bufio.NewReader(port)
			for {
				data, err := reader.ReadString('\n')
				if err != nil {
					log.Fatalf("Error reading from serial port: %v", err)
				}
				fmt.Print(data)
			}
		}()
		go func() {
			defer wg.Done()
			reader := bufio.NewReader(os.Stdin)
			for {
				input, err := reader.ReadString('\n')
				if err != nil {
					log.Fatalf("Error reading from stdin: %v", err)
				}
				input = strings.TrimSuffix(input, "\n")
				_, err = port.Write([]byte(input))
				if err != nil {
					log.Fatalf("Error writing to serial port: %v", err)
				}
			}
		}()
		wg.Wait()
	},
}

func init() {
	RootCmd.AddCommand(monCmd, sendCmd)
	monCmd.Flags().StringVarP(&ttyUSB, "ser", "m", "", "block device for serial interface (i.e. \"/dev/ttyACM0\")\nif unspecified serial connection will not be attempted")
	monCmd.Flags().IntVarP(&baud, "baud", "b", 9600, "baud rate")
	sendCmd.Flags().StringVarP(&ttyUSB, "ser", "m", "", "block device for serial interface (i.e. \"/dev/ttyACM0\")\nif unspecified serial connection will not be attempted")
	sendCmd.Flags().IntVarP(&baud, "baud", "b", 9600, "baud rate")
}

var monCmd = &cobra.Command{
	Use:                   "mon",
	Short:                 "serial monitor",
	Long:                  `serial monitor`,
	SilenceErrors:         true,
	SilenceUsage:          true,
	DisableSuggestions:    true,
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		port, err := serial.OpenPort(&serial.Config{Name: ttyUSB, Baud: baud})
		if err != nil {
			log.Fatalf("serial.OpenPort: %v", err)
		}
		defer port.Close()
		buf := make([]byte, 128)
		for {
			n, err := port.Read(buf)
			if err != nil {
				log.Fatalf("port.Read: %v", err)
			}
			fmt.Print(string(buf[:n]))
		}
	},
}

var sendCmd = &cobra.Command{
	Use:                   "send",
	Short:                 "serial send",
	Long:                  `serial send`,
	SilenceErrors:         true,
	SilenceUsage:          true,
	DisableSuggestions:    true,
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		port, err := serial.OpenPort(&serial.Config{Name: ttyUSB, Baud: baud})
		if err != nil {
			log.Fatalf("serial.OpenPort: %v", err)
		}
		defer port.Close()
		reader := bufio.NewReader(os.Stdin)
		for {
			input, err := reader.ReadString('\n')
			if err != nil {
				println("Error reading from stdin:", err)
				return
			}
			_, err = port.Write([]byte(input))
			if err != nil {
				log.Fatal(err)
			}
			time.Sleep(time.Millisecond * 100)
		}
	},
}

func init() {
	RootCmd.AddCommand(evalCmd)

	goProg = os.Getenv("GOPROG")
	if goProg != "" {

	fset = token.NewFileSet()
	var err error
	node, err = parser.ParseFile(fset, goProg, nil, parser.ParseComments)
	if err != nil {
		fmt.Println("Error parsing file:", err)
		os.Exit(1)
	}

	var commentAbove string



	for _, decl := range node.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.VAR {
			continue
		}

		if genDecl.Doc != nil && len(genDecl.Doc.List) > 0 {
			commentAbove = genDecl.Doc.List[0].Text
			} else {
				commentAbove = ""
			}

		for _, spec := range genDecl.Specs {
			valueSpec, ok := spec.(*ast.ValueSpec)
			if !ok || len(valueSpec.Values) == 0 {
				continue
			}

			if !isStringType(valueSpec.Type) {
				for _, name := range valueSpec.Names {
					jsonValue, err := getVar(node, name.Name)
					if err != nil {
						fmt.Printf("Error marshaling %s: %v\n", name.Name, err)
						continue
					}
					evalCmd.Flags().String(name.Name, jsonValue, strings.TrimSpace(commentAbove) + "\n\r\x1b[1;34m")
					efCmd.Flags().String(name.Name, jsonValue, strings.TrimSpace(commentAbove) + "\n\r\x1b[1;34m")
				}
			}
		}
	}
}
}

func init() {
	goProg = os.Getenv("GOPROG")

	if goProg != "" {
		fset := token.NewFileSet()

		file, err := parser.ParseFile(fset, goProg, nil, parser.ParseComments)
		if err != nil {
			fmt.Println("Error parsing Go file:", err)
			os.Exit(1)
		}

		var commentAbove string

		for _, decl := range file.Decls {
			genDecl, ok := decl.(*ast.GenDecl)
			if !ok || genDecl.Tok != token.VAR {
				continue
			}

			if genDecl.Doc != nil && len(genDecl.Doc.List) > 0 {
				commentAbove = genDecl.Doc.List[0].Text
			} else {
				commentAbove = ""
			}

			for _, spec := range genDecl.Specs {
				valueSpec := spec.(*ast.ValueSpec)
				for i, ident := range valueSpec.Names {
					if ident.Obj == nil {
						continue
					}
					varName := ident.Name
					flagName := fmt.Sprintf("%s", varName)
					flagDesc := ""
					if isStringType(valueSpec.Type) {
						flagDesc = fmt.Sprintf("main.%s", varName)
					}
					if commentAbove != "" {
						flagDesc += " " + strings.TrimSpace(commentAbove) + "\n\r\x1b[1;34m"
					}
					if i < len(valueSpec.Names) && valueSpec.Comment != nil {
						defaultValue := strings.TrimSpace(valueSpec.Comment.Text())
						if strings.HasPrefix(defaultValue, "//") {
							defaultValue = strings.TrimSpace(defaultValue[2:])
						}
						if !strings.Contains(flagDesc, "json") {
							defaultValue, _ = script.Exec(`bash -c 'printf ` + defaultValue + `' `).String()
						}
						if isStringType(valueSpec.Type) {
							flashCmd.Flags().String(flagName, defaultValue, flagDesc)
							efCmd.Flags().String(flagName, defaultValue, flagDesc)
						}
					}
				}
			}
		}
		_, err = script.Exec(`udisksctl help`).String()
		u := ""
		if err == nil {
			flashCmd.Flags().StringVarP(&ttyUSB, "ser", "m", "", "block device for serial interface (i.e. \"/dev/ttyACM0\")\nif unspecified serial connection will not be attempted")
			efCmd.Flags().StringVarP(&ttyUSB, "ser", "m", "", "block device for serial interface (i.e. \"/dev/ttyACM0\")\nif unspecified serial connection will not be attempted")
			flashCmd.Flags().IntVarP(&baud, "baud", "b", 9600, "baud rate")
			efCmd.Flags().IntVarP(&baud, "baud", "b", 9600, "baud rate")
			flashCmd.Flags().DurationVarP(&sleepTime, "slp", "s", 3*time.Second, "seconds to wait before serial connection after flashing")
			efCmd.Flags().DurationVarP(&sleepTime, "slp", "s", 3*time.Second, "seconds to wait before serial connection after flashing")
			flashCmd.Flags().StringVarP(&blkDev, "dev", "y", "", "block device to flash (i.e. \"/dev/sdx\")\nif unspecified, tinygo flash command is generated")
			efCmd.Flags().StringVarP(&blkDev, "dev", "y", "", "block device to flash (i.e. \"/dev/sdx\")\nif unspecified, tinygo flash command is generated")
		} else {
			u = "udisksctl not found ; mounting MCU block device not possible"
		}
		 cmdLong := fmt.Sprintf("\nGOPROG=%s %s \n%s", goProg, func() string {
			ret := ""
			if strings.HasPrefix(os.Args[0], "/tmp/go-build") {
				ret += " go run " + filepath.Base(os.Args[0]) + ".go "
			} else {
				ret += os.Args[0] + " "
			}
			for i, _ := range os.Args {
				if i > 0 {
					ret += os.Args[i] + " "
				}
			}
			return ret
		}(), u)
		flashCmd.Long += cmdLong
		flashCmd.Flags().StringVarP(&target, "target", "z", "", "tinygo flash target")
		efCmd.Long += cmdLong
		efCmd.Flags().StringVarP(&target, "target", "z", "", "tinygo flash target")
	} else {
		goFiles, _ := script.ListFiles("./").Match(".go").Slice()
		gofile := ""
		for _, f := range goFiles {
			if f == "main.go " {
				gofile = "main.go"
				break
			}
		}
		if gofile == "" && len(goFiles) > 0 {
			gofile = goFiles[0]
		} else {
			gofile = "/path/to/program.go"
		}

		cmdLong := fmt.Sprintf("\n\nGOPROG env not set\nGOPROG=%s %s \nGOPROG env should contain the the name of the .go program source file to compile and flash\nfor a menu of available flags set GOPROG env\n\nFor automatic flag generation with description and default values, use the format:\n\n//flag description\nvar SomeVar string //defaultvalue", gofile, func() string {
			ret := ""
			if strings.HasPrefix(os.Args[0], "/tmp/go-build") {
				ret += " go run " + filepath.Base(os.Args[0]) + ".go "
			} else {
				ret += os.Args[0] + " "
			}
			for i, _ := range os.Args {
				if i > 0 {
					ret += os.Args[i] + " "
				}
			}
			return ret
		}())
		flashCmd.Long += cmdLong
		efCmd.Long += cmdLong
	}
	_, err := script.Exec(`tinygo help`).String()
	if err == nil {
		RootCmd.AddCommand(flashCmd, efCmd)
	} else {
		script.Echo("tinygo not found ; flash subcommand not available\n").Stdout()
	}
	flashCmd.Flags().SortFlags = false
	efCmd.Flags().SortFlags = false
	evalCmd.Flags().SortFlags = false
}

func isStringType(expr ast.Expr) bool {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name == "string"
	case *ast.SelectorExpr:
		return false
	case *ast.ArrayType:
		return false
	default:
		return false
	}
}

func getVar(node *ast.File, varName string) (string, error) {
	for _, decl := range node.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.VAR {
			continue
		}

		for _, spec := range genDecl.Specs {
			valueSpec, ok := spec.(*ast.ValueSpec)
			if !ok || len(valueSpec.Values) == 0 {
				continue
			}

			for _, name := range valueSpec.Names {
				if name.Name == varName {
					var buf strings.Builder
					if err := printer.Fprint(&buf, token.NewFileSet(), valueSpec.Values[0]); err != nil {
						return "", err
					}

					return buf.String(), nil
				}
			}
		}
	}

	return "", fmt.Errorf("variable %s not found", varName)
}

var flashCmd = &cobra.Command{
	Use:   "flash",
	Short: "mcu cli - tinygo flash command generator",
	Long: `Tinygo application command line interface
Set global-scope string variables at compile time for tinygo applications
via flags autogenerated for this help menu.
` + time.Now().Format(time.RFC3339Nano),
	SilenceErrors:         true,
	SilenceUsage:          true,
	DisableSuggestions:    true,
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		if goProg == "" {
			fmt.Println("GOPROG not specified")
			os.Exit(0)
		}

		cmdToRun := "tinygo flash "
		if target != "" {
			cmdToRun += ` -target=` + target + ` `
		}
		var ldFlags string
		cmd.Flags().VisitAll(func(f *pflag.Flag) {
			if f.Value.Type() == "string" {
				if f.Value.String() != "" && strings.HasPrefix(f.Usage, "main.") && f.Name != "help" && f.Name != "ser" && f.Name != "dev" && f.Name != "target" {
					ldFlags += fmt.Sprintf(` -X 'main.%s=%s'`, f.Name, f.Value.String())
				}
			}
		})
		if ldFlags != "" {
			cmdToRun += ` -ldflags="` + ldFlags + `" `
		}
		cmdToRun += goProg
		if blkDev != "" {
			script.Echo(cmdToRun + "\n").Stdout()

			_, err := script.Exec(`bash -c  'set -x ; sudo echo "sudo cache" ;  set +x'`).Stdout()
			if err != nil {
				script.Echo(err.Error() + "\n").Stdout()
				os.Exit(1)
			}
			_, err = script.Exec(`bash -c  'set -x ; udisksctl mount -b ` + blkDev + ` ;  set +x'`).Stdout()
			if err != nil {
				script.Echo(err.Error() + "\n").Stdout()
				os.Exit(1)
			}
			_, err = script.Exec(`bash -c  '` + cmdToRun + `'`).Stdout()
			if err != nil {
				script.Echo(err.Error() + "\n").Stdout()
				os.Exit(1)
			}
		} else {
			script.Echo(cmdToRun + "\n").Stdout()
		}
		if ttyUSB != "" {
			var ttyusb string
			time.Sleep(sleepTime)
			var recursiveFunc func()
			recursiveFunc = func() {
				var err error
				ttyusb, err = script.Exec(`bash -c 'echo ` + ttyUSB + `'`).String()
				if err != nil {
					fmt.Println(err.Error())
					os.Exit(1)
				}
				hasQuestionMark := false
				for _, char := range ttyusb {
					if char == '?' {
						hasQuestionMark = true
						break
					}
				}
				if hasQuestionMark {
					time.Sleep(100 * time.Millisecond)
					recursiveFunc()
				} else {
					fmt.Printf("ttyusb found: %s\n", ttyusb)
					ttyusb = strings.TrimSpace(strings.ReplaceAll(ttyusb, "\n", ""))
				}
			}

			recursiveFunc()
			_, err := script.Exec(`bash -c  'sudo chmod a+rw  ` + ttyusb + `'`).Stdout()
			if err != nil {
				script.Echo(err.Error() + "\n").Stdout()
				os.Exit(1)
			}
			fmt.Printf("\"%s\"\n", ttyusb)
			var port *serial.Port
			for i := 0; i < 10; i++ {

				port, err = serial.OpenPort(&serial.Config{Name: ttyusb, Baud: baud})
				if err == nil {
					break
				}
				log.Printf("serial.OpenPort: %v\n", err)
				time.Sleep(time.Second)
			}
			if err != nil {
				log.Printf("serial.OpenPort: %v\n", err)
			}
			defer port.Close()
			var wg sync.WaitGroup
			wg.Add(2)
			go func() {
				defer wg.Done()
				reader := bufio.NewReader(port)
				for {
					data, err := reader.ReadString('\n')
					if err != nil {
						log.Fatalf("Error reading from serial port: %v", err)
					}
					fmt.Print(data)
				}
			}()
			go func() {
				defer wg.Done()
				reader := bufio.NewReader(os.Stdin)
				for {
					input, err := reader.ReadString('\n')
					if err != nil {
						log.Fatalf("Error reading from stdin: %v", err)
					}
					input = strings.TrimSuffix(input, "\n") // Trim newline character
					_, err = port.Write([]byte(input))
					if err != nil {
						log.Fatalf("Error writing to serial port: %v", err)
					}
				}
			}()
			wg.Wait()
		}
	},
}

var evalCmd = &cobra.Command{
	Use:   "eval",
	Short: "generate updated source code",
	Long: `modify initializations in source code for tinygo applications
` + time.Now().Format(time.RFC3339Nano),
	SilenceErrors:         true,
	SilenceUsage:          true,
	DisableSuggestions:    true,
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		if goProg == "" {
			fmt.Println("GOPROG not specified")
			os.Exit(1)
		}
		ast.Inspect(node, func(n ast.Node) bool {
			if vs, ok := n.(*ast.ValueSpec); ok {
				for _, name := range vs.Names {
					fn := name.Name
					if fn == "help" ||  fn == "ser" ||  fn == "baud" ||  fn == "slp" ||  fn == "target" ||  fn == "dev" {
						continue
					}
					flagValue, err := cmd.Flags().GetString(name.Name)
					if err == nil {
						vs.Values = []ast.Expr{&ast.BasicLit{
							Kind:  token.STRING,
							Value: flagValue,
						}}
					}
				}
			}
			return true
		})

		var buf bytes.Buffer
		if err := printer.Fprint(&buf, fset, node); err != nil {
			log.Fatalf("Failed to print source: %v", err)
		}
		fmt.Println(buf.String())
	},
}

var efCmd = &cobra.Command{
	Use:   "ef",
	Short: "eval + flash",
	Long: `seamlessly flash modified source code
` + time.Now().Format(time.RFC3339Nano),
	SilenceErrors:         true,
	SilenceUsage:          true,
	DisableSuggestions:    true,
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		if goProg == "" {
			fmt.Println("GOPROG not specified")
			os.Exit(1)
		}
		ast.Inspect(node, func(n ast.Node) bool {
			if vs, ok := n.(*ast.ValueSpec); ok {
				for _, name := range vs.Names {
					fn := name.Name
					if fn == "help" ||  fn == "ser" ||  fn == "baud" ||  fn == "slp" ||  fn == "target" ||  fn == "dev" {
						continue
					}
					fv, err := cmd.Flags().GetString(fn)
					if err == nil && !strings.HasPrefix(cmd.Flags().Lookup(fn).Usage, "main.") {
						vs.Values = []ast.Expr{&ast.BasicLit{
							Kind:  token.STRING,
							Value: fv,
						}}
					}
				}
			}
			return true
		})

		var buf bytes.Buffer
		if err := printer.Fprint(&buf, fset, node); err != nil {
			log.Fatalf("Failed to print source: %v", err)
		}
//		fmt.Println(buf.String())

		tempFile, err := os.CreateTemp(os.TempDir(), "*.go")
		if err != nil {
	        fmt.Println("Error creating temporary file:", err)
	        return
	    }
	    _, err = tempFile.Write(buf.Bytes())
	    if err != nil {
	        fmt.Println("Error writing to temporary file:", err)
	        return
	    }
		tempFile.Close()
		cmdToRun := "tinygo flash "
		if target != "" {
			cmdToRun += ` -target=` + target + ` `
		}
		var ldFlags string
		cmd.Flags().VisitAll(func(f *pflag.Flag) {
			if f.Value.Type() == "string" {
				if f.Value.String() != "" && strings.HasPrefix(f.Usage, "main.") && f.Name != "help" && f.Name != "ser" && f.Name != "dev" && f.Name != "target" {
					ldFlags += fmt.Sprintf(` -X 'main.%s=%s'`, f.Name, f.Value.String())
				}
			}
		})
		if ldFlags != "" {
			cmdToRun += ` -ldflags="` + ldFlags + `" `
		}
		cmdToRun += tempFile.Name()
		if blkDev != "" {
			script.Echo(cmdToRun + "\n").Stdout()

			_, err := script.Exec(`bash -c  'set -x ; sudo echo "sudo cache" ;  set +x'`).Stdout()
			if err != nil {
				script.Echo(err.Error() + "\n").Stdout()
				os.Exit(1)
			}
			_, err = script.Exec(`bash -c  'set -x ; udisksctl mount -b ` + blkDev + ` ;  set +x'`).Stdout()
			if err != nil {
				script.Echo(err.Error() + "\n").Stdout()
				os.Exit(1)
			}
			_, err = script.Exec(`bash -c  '` + cmdToRun + `'`).Stdout()
			if err != nil {
				script.Echo(err.Error() + "\n").Stdout()
				os.Exit(1)
			}

			} else {
				script.Echo(cmdToRun + "\n").Stdout()
			}
			os.Remove(tempFile.Name())
			if ttyUSB != "" {
				var ttyusb string
				time.Sleep(sleepTime)
				var recursiveFunc func()
				recursiveFunc = func() {
					var err error
					ttyusb, err = script.Exec(`bash -c 'echo ` + ttyUSB + `'`).String()
					if err != nil {
						fmt.Println(err.Error())
						os.Exit(1)
					}
					hasQuestionMark := false
					for _, char := range ttyusb {
						if char == '?' {
							hasQuestionMark = true
							break
						}
					}
					if hasQuestionMark {
						time.Sleep(100 * time.Millisecond)
						recursiveFunc()
						} else {
							fmt.Printf("ttyusb found: %s\n", ttyusb)
							ttyusb = strings.TrimSpace(strings.ReplaceAll(ttyusb, "\n", ""))
						}
					}
					recursiveFunc()
					_, err := script.Exec(`bash -c  'sudo chmod a+rw  ` + ttyusb + `'`).Stdout()
					if err != nil {
						script.Echo(err.Error() + "\n").Stdout()
						os.Exit(1)
					}
					fmt.Printf("\"%s\"\n", ttyusb)
					var port *serial.Port
					for i := 0; i < 10; i++ {
						port, err = serial.OpenPort(&serial.Config{Name: ttyusb, Baud: baud})
						if err == nil {
							break
						}
						log.Printf("serial.OpenPort: %v\n", err)
						time.Sleep(time.Second)
					}
					if err != nil {
						log.Printf("serial.OpenPort: %v\n", err)
					}
					defer port.Close()
					var wg sync.WaitGroup
					wg.Add(2)
					go func() {
						defer wg.Done()
						reader := bufio.NewReader(port)
						for {
							data, err := reader.ReadString('\n')
							if err != nil {
								log.Fatalf("Error reading from serial port: %v", err)
							}
							fmt.Print(data)
						}
						}()
						go func() {
							defer wg.Done()
							reader := bufio.NewReader(os.Stdin)
							for {
								input, err := reader.ReadString('\n')
								if err != nil {
									log.Fatalf("Error reading from stdin: %v", err)
								}
								input = strings.TrimSuffix(input, "\n")
								_, err = port.Write([]byte(input))
								if err != nil {
									log.Fatalf("Error writing to serial port: %v", err)
								}
							}
							}()
							wg.Wait()
						}

	},
}



const help = "Usage:\r\n" +
	"  {{.UseLine}}{{if .HasAvailableSubCommands}}{{end}} {{if gt (len .Aliases) 0}}\r\n\r\n" +
	"{{.NameAndAliases}}{{end}}{{if .HasAvailableSubCommands}}\r\n\r\n" +
	"Available Commands:{{range .Commands}}{{if (or .IsAvailableCommand)}}\r\n  " +
	"{{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}\r\n\r\n" +
	"Flags:\r\n" +
	"{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}\r\n\r\n" +
	"Global Flags:\r\n" +
	"{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}\r\n\r\n\033[0m"
