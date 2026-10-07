package command

import (
	"regexp"
	"strconv"
	"strings"
)

type opts struct {
	format, sep                  string
	channel                      string
	days                         *int
	noProject, noDate, clipboard bool
	separatorSet                 bool
	channelSet                   bool
	pos                          []string
}

func parseOpts(args []string, allowed map[string]bool) (opts, error) {
	o := opts{format: "csv", sep: "\t"}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			o.pos = append(o.pos, args[i+1:]...)
			break
		}
		key, val, has := arg, "", false
		if strings.HasPrefix(arg, "--") && strings.Contains(arg, "=") {
			key, val, has = strings.Cut(arg, "=")
		} else if len(arg) > 2 && arg[0] == '-' && strings.ContainsRune("fsd", rune(arg[1])) {
			key, val, has = arg[:2], arg[2:], true
		}
		take := func() (string, error) {
			if has {
				return val, nil
			}
			if i+1 >= len(args) {
				return "", usagef("option %s requires a value", key)
			}
			if len(args[i+1]) > 1 && strings.HasPrefix(args[i+1], "-") {
				return "", usagef("option %s requires a value; use a long option with = for values starting with a dash", key)
			}
			i++
			return args[i], nil
		}
		switch key {
		case "-f", "--format":
			if !allowed["format"] {
				return o, usagef("unknown option: %s", key)
			}
			v, e := take()
			if e != nil {
				return o, e
			}
			o.format = v
		case "-s", "--separator":
			if !allowed["separator"] {
				return o, usagef("unknown option: %s", key)
			}
			v, e := take()
			if e != nil {
				return o, e
			}
			o.sep = v
			o.separatorSet = true
		case "-d", "--days":
			if !allowed["days"] {
				return o, usagef("unknown option: %s", key)
			}
			v, e := take()
			if e != nil {
				return o, e
			}
			n, ok := parseUnsignedInteger(v)
			if !ok {
				return o, usage("days must be a non-negative integer")
			}
			o.days = &n
		case "--channel":
			if !allowed["channel"] {
				return o, usagef("unknown option: %s", key)
			}
			v, e := take()
			if e != nil {
				return o, e
			}
			o.channel, o.channelSet = v, true
		case "--no-project":
			if !allowed["no-project"] {
				return o, usagef("unknown option: %s", key)
			}
			if has {
				return o, usagef("Option '%s' does not take an argument", key)
			}
			o.noProject = true
		case "--no-date":
			if !allowed["no-date"] {
				return o, usagef("unknown option: %s", key)
			}
			if has {
				return o, usagef("Option '%s' does not take an argument", key)
			}
			o.noDate = true
		case "--clipboard":
			if !allowed["clipboard"] {
				return o, usagef("unknown option: %s", key)
			}
			if has {
				return o, usagef("Option '%s' does not take an argument", key)
			}
			o.clipboard = true
		default:
			if strings.HasPrefix(arg, "-") {
				return o, usagef("unknown option: %s", arg)
			}
			o.pos = append(o.pos, arg)
		}
	}
	if o.format != "csv" && o.format != "json" && o.format != "table" {
		return o, usage("format must be csv, json, or table")
	}
	return o, nil
}

func parseUnsignedInteger(value string) (int, bool) {
	if !regexp.MustCompile(`^\d+$`).MatchString(value) {
		return 0, false
	}
	n, err := strconv.Atoi(value)
	// Preserve the integer range accepted by the previous JavaScript CLI.
	return n, err == nil && int64(n) <= 9007199254740991
}
