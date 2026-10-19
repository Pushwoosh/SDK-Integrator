// Command pushwoosh-integrator wires the Pushwoosh SDK into a project with one install code.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/Pushwoosh/SDK-Integrator/internal/agent"
	"github.com/Pushwoosh/SDK-Integrator/internal/recipe"
	"github.com/Pushwoosh/SDK-Integrator/internal/redeem"
	"github.com/Pushwoosh/SDK-Integrator/internal/ui"
	"github.com/Pushwoosh/SDK-Integrator/internal/web"
)

var version = "dev"

var recipes = []recipe.Recipe{web.Recipe{}}

func main() {
	fs := flag.NewFlagSet("pushwoosh-integrator", flag.ExitOnError)
	api := fs.String("api", redeem.DefaultAPI, "Pushwoosh API host that issued the code")
	dir := fs.String("dir", ".", "project directory")
	yes := fs.Bool("yes", false, "apply without asking")
	dryRun := fs.Bool("dry-run", false, "show the plan and exit; the code stays unused")
	force := fs.Bool("force", false, "run even with uncommitted changes")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: pushwoosh-integrator [flags] <install-code> [claude|cursor]\n       pushwoosh-integrator redeem <install-code>\n\n")
		fs.PrintDefaults()
	}

	args := os.Args[1:]
	if len(args) > 0 && args[0] == "version" {
		fmt.Println(version)

		return
	}
	redeemOnly := len(args) > 0 && args[0] == "redeem"
	if redeemOnly {
		args = args[1:]
	}
	_ = fs.Parse(args)
	if fs.NArg() < 1 {
		fs.Usage()
		os.Exit(2)
	}
	code := fs.Arg(0)
	ctx := context.Background()

	if redeemOnly {
		creds, err := redeem.Redeem(ctx, *api, code)
		if err != nil {
			ui.Fail("%v", err)
		}
		_ = json.NewEncoder(os.Stdout).Encode(creds)

		return
	}

	if fs.NArg() > 1 {
		next, err := agent.Install(fs.Arg(1), *dir, code)
		if err != nil {
			ui.Fail("%v", err)
		}
		ui.Say("\n%s", next)

		return
	}

	if !*force && dirty(*dir) {
		ui.Fail("the project has uncommitted changes; commit or stash them so the install is one clean diff, or pass --force")
	}

	var rcp recipe.Recipe
	for _, r := range recipes {
		if r.Detect(*dir) {
			rcp = r

			break
		}
	}
	if rcp == nil {
		ui.Fail("no supported project found in %s; run with an AI agent instead: ... | sh -s -- %s claude", *dir, code)
	}

	// The plan is built with placeholder credentials so a dry run or a refusal leaves the code unspent.
	placeholder := recipe.Credentials{ApplicationCode: "XXXXX-XXXXX", APIToken: "<device API token>"}
	plan, err := rcp.Plan(*dir, placeholder)
	if errors.Is(err, web.ErrAlreadyIntegrated) {
		ui.Say("%v; nothing to do.", err)

		return
	}
	if err != nil {
		ui.Fail("%v", err)
	}

	show(plan)
	if *dryRun {
		return
	}
	if !*yes && !ui.Confirm("Apply these changes?") {
		ui.Say("Nothing changed. The install code is still valid until it expires.")

		return
	}

	creds, err := redeem.Redeem(ctx, *api, code)
	if err != nil {
		ui.Fail("%v", err)
	}
	plan, err = rcp.Plan(*dir, recipe.Credentials{ApplicationCode: creds.ApplicationCode, APIToken: creds.APIToken})
	if err != nil {
		ui.Fail("%v", err)
	}
	if err := recipe.Apply(*dir, plan); err != nil {
		ui.Fail("%v", err)
	}
	if len(plan.Command) > 0 {
		ui.Say("==> %s", strings.Join(plan.Command, " "))
		cmd := exec.Command(plan.Command[0], plan.Command[1:]...)
		cmd.Dir, cmd.Stdout, cmd.Stderr = *dir, os.Stdout, os.Stderr
		if err := cmd.Run(); err != nil {
			ui.Fail("the code is in place, but %q failed: %v; run it yourself", strings.Join(plan.Command, " "), err)
		}
	}

	ui.Say("\nPushwoosh is integrated (application %s).", creds.ApplicationCode)
	ui.Say("Next: start your site, open it in the browser and allow notifications; the device shows up in the Control Panel.")
	for _, n := range plan.Notes {
		ui.Say("Note: %s", n)
	}
}

func show(p *recipe.Plan) {
	ui.Say("Detected: %s\n", p.Stack)
	for _, c := range p.Changes {
		fmt.Println(recipe.Diff(c))
	}
	if len(p.Command) > 0 {
		ui.Say("Then runs: %s\n", strings.Join(p.Command, " "))
	}
	ui.Say("XXXXX-XXXXX and the token are filled in from your install code once you confirm.")
}

// dirty is false outside a git checkout: there is nothing to compare against.
func dirty(dir string) bool {
	out, err := exec.Command("git", "-C", dir, "status", "--porcelain").Output()

	return err == nil && len(strings.TrimSpace(string(out))) > 0
}
