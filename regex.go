package goproject

import "regexp"

type regexStat struct {
	pattern, before string
	match, after    string
	start, length   int
	regex           *regexp.Regexp
}

//─────────────┤ mapFromPatSlice ├─────────────

func mapFromPatSlice(pats []string) map[string]*regexp.Regexp {
	var regs = make(map[string]*regexp.Regexp)
	for _, p := range pats {
		regs[p] = regexp.MustCompile(p)
	}

	return regs
}

//─────────────┤ regexStatFromPat ├─────────────

func regexStatFromPat(pat, search string) regexStat {
	if len(search) == 0 || len(pat) == 0 {
		return regexStat{}
	}

	r := regexp.MustCompile(pat)
	loc := r.FindStringIndex(search)

	if loc == nil {
		return regexStat{pattern: pat, before: search}
	}

	bef := search[:loc[0]]
	m := search[loc[0]:loc[1]]
	aft := search[loc[1]:]

	return regexStat{
		pattern: pat,
		before:  bef,
		match:   m,
		after:   aft,
		start:   loc[0],
		length:  loc[1] - loc[0],
		regex:   r,
	}
}


