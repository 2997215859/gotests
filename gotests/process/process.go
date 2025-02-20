// Package process is a thin wrapper around the gotests library. It is intended
// to be called from a binary and handle its arguments, flags, and output when
// generating tests.
package process

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"regexp"

	"github.com/2997215859/gotests"
)

const newFilePerm os.FileMode = 0644

const (
	specifyFlagMessage = "Please specify either the -only, -excl, -exported, or -all flag"
	specifyFileMessage = "Please specify a file or directory containing the source"
)

// Set of options to use when generating tests.
type Options struct {
	OnlyFuncs          string   // Regexp string for filter matches.
	ExclFuncs          string   // Regexp string for excluding matches.
	ExportedFuncs      bool     // Only include exported functions.
	AllFuncs           bool     // Include all non-tested functions.
	PrintInputs        bool     // Print function parameters as part of error messages.
	Subtests           bool     // Print tests using Go 1.7 subtests
	Parallel           bool     // Print tests that runs the subtests in parallel.
	Named              bool     // Create Map instead of slice
	WriteOutput        bool     // Write output to test file(s).
	Template           string   // Name of custom template set
	TemplateDir        string   // Path to custom template set
	TemplateParamsPath string   // Path to custom parameters json file(s).
	TemplateData       [][]byte // Data slice for templates
}

// Generates tests for the Go files defined in args with the given options.
// Logs information and errors to out. By default outputs generated tests to
// out unless specified by opt.
func Run(out io.Writer, args []string, opts *Options) {
	if opts == nil {
		opts = &Options{}
	}
	opt := parseOptions(out, opts)
	if opt == nil {
		return
	}
	if len(args) == 0 {
		fmt.Fprintln(out, specifyFileMessage)
		return
	}
	for _, path := range args {
		generateTests(out, path, opts.WriteOutput, opt)
	}
}

func parseOptions(out io.Writer, opt *Options) *gotests.Options {
	if opt.OnlyFuncs == "" && opt.ExclFuncs == "" && !opt.ExportedFuncs && !opt.AllFuncs {
		fmt.Fprintln(out, specifyFlagMessage)
		return nil
	}
	onlyRE, err := parseRegexp(opt.OnlyFuncs)
	if err != nil {
		fmt.Fprintln(out, "Invalid -only regex:", err)
		return nil
	}
	exclRE, err := parseRegexp(opt.ExclFuncs)
	if err != nil {
		fmt.Fprintln(out, "Invalid -excl regex:", err)
		return nil
	}

	templateParams := map[string]interface{}{}
	jfile := opt.TemplateParamsPath
	if jfile != "" {
		buf, err := ioutil.ReadFile(jfile)
		if err != nil {
			fmt.Fprintf(out, "Failed to read from %s ,err %s", jfile, err)
			return nil
		}

		err = json.Unmarshal(buf, &templateParams)
		if err != nil {
			fmt.Fprintf(out, "Failed to umarshal %s er %s", jfile, err)
			return nil
		}
	}

	return &gotests.Options{
		Only:           onlyRE,
		Exclude:        exclRE,
		Exported:       opt.ExportedFuncs,
		PrintInputs:    opt.PrintInputs,
		Subtests:       opt.Subtests,
		Parallel:       opt.Parallel,
		Named:          opt.Named,
		Template:       opt.Template,
		TemplateDir:    opt.TemplateDir,
		TemplateParams: templateParams,
		TemplateData:   opt.TemplateData,
	}
}

func parseRegexp(s string) (*regexp.Regexp, error) {
	if s == "" {
		return nil, nil
	}
	re, err := regexp.Compile(s)
	if err != nil {
		return nil, err
	}
	return re, nil
}

func generateTests(out io.Writer, path string, writeOutput bool, opt *gotests.Options) {
	gts, err := gotests.GenerateTests(path, opt)
	if err != nil {
		fmt.Fprintln(out, err.Error())
		return
	}
	if len(gts) == 0 {
		fmt.Fprintln(out, "No tests generated for", path)
		return
	}
	for _, t := range gts {
		outputTest(out, t, writeOutput)
	}
}

func outputTest(out io.Writer, t *gotests.GeneratedTest, writeOutput bool) {
	if writeOutput {
		if err := ioutil.WriteFile(t.Path, t.Output, newFilePerm); err != nil {
			fmt.Fprintln(out, err)
			return
		}
	}
	for _, t := range t.Functions {
		fmt.Fprintln(out, "Generated", t.TestName())
	}
	if !writeOutput {
		out.Write(t.Output)
	}
}
