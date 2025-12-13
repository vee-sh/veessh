package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestProfileValidate(t *testing.T) {
	tests := []struct {
		name    string
		profile Profile
		wantErr bool
	}{
		{
			name: "valid SSH profile",
			profile: Profile{
				Name:     "test",
				Protocol: ProtocolSSH,
				Host:     "example.com",
			},
			wantErr: false,
		},
		{
			name: "valid SFTP profile",
			profile: Profile{
				Name:     "test",
				Protocol: ProtocolSFTP,
				Host:     "example.com",
			},
			wantErr: false,
		},
		{
			name: "valid Telnet profile",
			profile: Profile{
				Name:     "test",
				Protocol: ProtocolTelnet,
				Host:     "example.com",
			},
			wantErr: false,
		},
		{
			name: "valid Mosh profile",
			profile: Profile{
				Name:     "test",
				Protocol: ProtocolMosh,
				Host:     "example.com",
			},
			wantErr: false,
		},
		{
			name: "valid SSM profile",
			profile: Profile{
				Name:       "test",
				Protocol:   ProtocolSSM,
				Host:       "i-1234567890",
				InstanceID: "i-1234567890",
			},
			wantErr: false,
		},
		{
			name: "valid GCloud profile",
			profile: Profile{
				Name:     "test",
				Protocol: ProtocolGCloud,
				Host:     "my-vm",
			},
			wantErr: false,
		},
		{
			name: "missing name",
			profile: Profile{
				Protocol: ProtocolSSH,
				Host:     "example.com",
			},
			wantErr: true,
		},
		{
			name: "missing host",
			profile: Profile{
				Name:     "test",
				Protocol: ProtocolSSH,
			},
			wantErr: true,
		},
		{
			name: "invalid protocol",
			profile: Profile{
				Name:     "test",
				Protocol: "invalid",
				Host:     "example.com",
			},
			wantErr: true,
		},
		{
			name: "whitespace name",
			profile: Profile{
				Name:     "   ",
				Protocol: ProtocolSSH,
				Host:     "example.com",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := (&tt.profile).Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestProfileValidateSetsPorts(t *testing.T) {
	tests := []struct {
		protocol Protocol
		wantPort int
	}{
		{ProtocolSSH, 22},
		{ProtocolSFTP, 22},
		{ProtocolMosh, 22},
		{ProtocolTelnet, 23},
	}

	for _, tt := range tests {
		t.Run(string(tt.protocol), func(t *testing.T) {
			p := Profile{
				Name:     "test",
				Protocol: tt.protocol,
				Host:     "example.com",
			}
			if err := (&p).Validate(); err != nil {
				t.Fatalf("Validate() error = %v", err)
			}
			if p.Port != tt.wantPort {
				t.Errorf("Port = %d, want %d", p.Port, tt.wantPort)
			}
		})
	}
}

func TestConfigCRUD(t *testing.T) {
	cfg := Config{Profiles: map[string]Profile{}}

	// Test UpsertProfile
	p1 := Profile{Name: "test1", Protocol: ProtocolSSH, Host: "host1.com"}
	cfg.UpsertProfile(p1)

	if len(cfg.Profiles) != 1 {
		t.Errorf("Expected 1 profile, got %d", len(cfg.Profiles))
	}

	// Test GetProfile
	got, ok := cfg.GetProfile("test1")
	if !ok {
		t.Error("GetProfile should return true for existing profile")
	}
	if got.Host != "host1.com" {
		t.Errorf("Host = %s, want host1.com", got.Host)
	}

	// Test GetProfile not found
	_, ok = cfg.GetProfile("nonexistent")
	if ok {
		t.Error("GetProfile should return false for non-existent profile")
	}

	// Test update via UpsertProfile
	p1Updated := Profile{Name: "test1", Protocol: ProtocolSSH, Host: "updated.com"}
	cfg.UpsertProfile(p1Updated)
	got, _ = cfg.GetProfile("test1")
	if got.Host != "updated.com" {
		t.Errorf("Host = %s, want updated.com", got.Host)
	}

	// Test DeleteProfile
	deleted := cfg.DeleteProfile("test1")
	if !deleted {
		t.Error("DeleteProfile should return true for existing profile")
	}
	if len(cfg.Profiles) != 0 {
		t.Error("Profile should be deleted")
	}

	// Test DeleteProfile not found
	deleted = cfg.DeleteProfile("nonexistent")
	if deleted {
		t.Error("DeleteProfile should return false for non-existent profile")
	}
}

func TestConfigListProfiles(t *testing.T) {
	cfg := Config{Profiles: map[string]Profile{
		"z-profile": {Name: "z-profile", Group: "b-group"},
		"a-profile": {Name: "a-profile", Group: "b-group"},
		"m-profile": {Name: "m-profile", Group: "a-group"},
	}}

	list := cfg.ListProfiles()
	if len(list) != 3 {
		t.Fatalf("Expected 3 profiles, got %d", len(list))
	}

	// Should be sorted by group, then name
	expected := []string{"m-profile", "a-profile", "z-profile"}
	for i, p := range list {
		if p.Name != expected[i] {
			t.Errorf("Profile %d = %s, want %s", i, p.Name, expected[i])
		}
	}
}

func TestProfileInheritance(t *testing.T) {
	cfg := Config{Profiles: map[string]Profile{
		"parent": {
			Name:         "parent",
			Protocol:     ProtocolSSH,
			Host:         "parent.example.com",
			Port:         22,
			Username:     "parentuser",
			IdentityFile: "/path/to/key",
			Group:        "production",
			Tags:         []string{"prod", "web"},
		},
		"child": {
			Name:     "child",
			Host:     "child.example.com",
			Username: "childuser",
			Extends:  "parent",
		},
	}}

	child, ok := cfg.GetProfile("child")
	if !ok {
		t.Fatal("Child profile not found")
	}

	// Should inherit from parent
	if child.Protocol != ProtocolSSH {
		t.Errorf("Protocol = %s, want ssh (inherited)", child.Protocol)
	}
	if child.IdentityFile != "/path/to/key" {
		t.Errorf("IdentityFile = %s, want /path/to/key (inherited)", child.IdentityFile)
	}
	if child.Group != "production" {
		t.Errorf("Group = %s, want production (inherited)", child.Group)
	}

	// Should override parent values
	if child.Host != "child.example.com" {
		t.Errorf("Host = %s, want child.example.com", child.Host)
	}
	if child.Username != "childuser" {
		t.Errorf("Username = %s, want childuser", child.Username)
	}

	// Name should always be child's
	if child.Name != "child" {
		t.Errorf("Name = %s, want child", child.Name)
	}

	// Extends should be cleared after resolution
	if child.Extends != "" {
		t.Errorf("Extends = %s, should be empty after resolution", child.Extends)
	}
}

func TestProfileInheritanceChain(t *testing.T) {
	cfg := Config{Profiles: map[string]Profile{
		"grandparent": {
			Name:         "grandparent",
			Protocol:     ProtocolSSH,
			Host:         "gp.example.com",
			IdentityFile: "/path/to/gp-key",
		},
		"parent": {
			Name:     "parent",
			Host:     "parent.example.com",
			Username: "parentuser",
			Extends:  "grandparent",
		},
		"child": {
			Name:    "child",
			Host:    "child.example.com",
			Extends: "parent",
		},
	}}

	child, ok := cfg.GetProfile("child")
	if !ok {
		t.Fatal("Child profile not found")
	}

	// Should inherit through chain
	if child.Protocol != ProtocolSSH {
		t.Errorf("Protocol = %s, want ssh (from grandparent)", child.Protocol)
	}
	if child.IdentityFile != "/path/to/gp-key" {
		t.Errorf("IdentityFile = %s, want /path/to/gp-key (from grandparent)", child.IdentityFile)
	}
	if child.Username != "parentuser" {
		t.Errorf("Username = %s, want parentuser (from parent)", child.Username)
	}
	if child.Host != "child.example.com" {
		t.Errorf("Host = %s, want child.example.com", child.Host)
	}
}

func TestProfileInheritanceCycleProtection(t *testing.T) {
	cfg := Config{Profiles: map[string]Profile{
		"a": {Name: "a", Protocol: ProtocolSSH, Host: "a.com", Extends: "b"},
		"b": {Name: "b", Protocol: ProtocolSSH, Host: "b.com", Extends: "a"},
	}}

	// Should not infinite loop
	_, ok := cfg.GetProfile("a")
	if !ok {
		t.Error("GetProfile should still work with cyclic inheritance")
	}
}

func TestSaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.yaml")

	original := Config{Profiles: map[string]Profile{
		"test": {
			Name:            "test",
			Protocol:        ProtocolSSH,
			Host:            "example.com",
			Port:            22,
			Username:        "user",
			IdentityFile:    "/path/to/key",
			UseAgent:        true,
			Group:           "prod",
			Description:     "Test profile",
			Favorite:        true,
			LastUsed:        time.Now().Truncate(time.Second),
			UseCount:        5,
			ProxyJump:       "jump.example.com",
			Tags:            []string{"web", "prod"},
			LocalForwards:   []string{"8080:localhost:80"},
			RemoteForwards:  []string{"9090:localhost:90"},
			DynamicForwards: []string{"1080"},
			RemoteCommand:   "tmux attach",
			RemoteDir:       "/app",
		},
	}}

	// Save
	if err := Save(cfgPath, original); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Check file permissions
	info, err := os.Stat(cfgPath)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("File permissions = %04o, want 0600", perm)
	}

	// Load
	loaded, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Verify
	got, ok := loaded.GetProfile("test")
	if !ok {
		t.Fatal("Profile not found after load")
	}
	want := original.Profiles["test"]

	if got.Name != want.Name {
		t.Errorf("Name = %s, want %s", got.Name, want.Name)
	}
	if got.Host != want.Host {
		t.Errorf("Host = %s, want %s", got.Host, want.Host)
	}
	if got.Username != want.Username {
		t.Errorf("Username = %s, want %s", got.Username, want.Username)
	}
	if got.UseCount != want.UseCount {
		t.Errorf("UseCount = %d, want %d", got.UseCount, want.UseCount)
	}
	if len(got.Tags) != len(want.Tags) {
		t.Errorf("Tags length = %d, want %d", len(got.Tags), len(want.Tags))
	}
}

func TestLoadNonExistent(t *testing.T) {
	cfg, err := Load("/nonexistent/path/config.yaml")
	if err != nil {
		t.Errorf("Load() should not error for non-existent file, got %v", err)
	}
	if cfg.Profiles == nil {
		t.Error("Profiles map should be initialized")
	}
	if len(cfg.Profiles) != 0 {
		t.Error("Profiles map should be empty")
	}
}

func TestLoadInvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.yaml")

	if err := os.WriteFile(cfgPath, []byte("invalid: yaml: content: ["), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := Load(cfgPath)
	if err == nil {
		t.Error("Load() should error for invalid YAML")
	}
}

func BenchmarkGetProfile(b *testing.B) {
	cfg := Config{Profiles: make(map[string]Profile)}
	for i := 0; i < 100; i++ {
		name := "profile" + string(rune('0'+i%10)) + string(rune('0'+i/10))
		cfg.Profiles[name] = Profile{Name: name, Protocol: ProtocolSSH, Host: "example.com"}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cfg.GetProfile("profile55")
	}
}

func BenchmarkGetProfileWithInheritance(b *testing.B) {
	cfg := Config{Profiles: map[string]Profile{
		"parent": {Name: "parent", Protocol: ProtocolSSH, Host: "parent.com", IdentityFile: "/key"},
		"child":  {Name: "child", Host: "child.com", Extends: "parent"},
	}}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cfg.GetProfile("child")
	}
}

func BenchmarkListProfiles(b *testing.B) {
	cfg := Config{Profiles: make(map[string]Profile)}
	for i := 0; i < 100; i++ {
		name := "profile" + string(rune('0'+i%10)) + string(rune('0'+i/10))
		cfg.Profiles[name] = Profile{Name: name, Group: "group" + string(rune('0'+i%5))}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cfg.ListProfiles()
	}
}

func BenchmarkValidate(b *testing.B) {
	p := Profile{
		Name:     "test",
		Protocol: ProtocolSSH,
		Host:     "example.com",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		(&p).Validate()
	}
}

