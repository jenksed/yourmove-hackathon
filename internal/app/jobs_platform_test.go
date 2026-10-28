package app

import "testing"

func TestJobPlatformMapping(t *testing.T) {
	cases := []struct {
		host  string
		key   string
		label string
	}{
		{"jobs.ashbyhq.com", "ashby", "Ashby"},
		{"boards.greenhouse.io", "greenhouse", "Greenhouse"},
		{"job-boards.greenhouse.io", "greenhouse", "Greenhouse"},
		{"jobs.lever.co", "lever", "Lever"},
		{"jobs.eu.lever.co", "lever", "Lever"},
		{"jobs.smartrecruiters.com", "smartrecruiters", "SmartRecruiters"},
		{"apply.workable.com", "workable", "Workable"},
		{"jobs.jobvite.com", "jobvite", "Jobvite"},
		{"acme.myworkdayjobs.com", "workday", "Workday"},
		{"acme.teamtailor.com", "teamtailor", "Teamtailor"},
		{"acme.recruitee.com", "recruitee", "Recruitee"},
		{"acme.icims.com", "icims", "iCIMS"},
		{"acme.jobs.personio.de", "personio", "Personio"},
		{"acme.jobs.personio.com", "personio", "Personio"},
		{"2000recruiting.paylocity.com", "paylocity", "Paylocity"},
		{"Jobs.AshbyHQ.com", "ashby", "Ashby"},
		{"careers.example.com", "", ""},
		{"jobs.ashbyhq.com.attacker.example", "", ""},
		{"", "", ""},
	}
	for _, tc := range cases {
		j := Job{Host: tc.host}
		if got := j.PlatformKey(); got != tc.key {
			t.Errorf("PlatformKey(%q) = %q, want %q", tc.host, got, tc.key)
		}
		if got := j.PlatformLabel(); got != tc.label {
			t.Errorf("PlatformLabel(%q) = %q, want %q", tc.host, got, tc.label)
		}
		if tc.key != "" && j.PlatformGlyph() == "" {
			t.Errorf("PlatformGlyph(%q) is empty", tc.host)
		}
	}
}
