package toml

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestConfLoader(t *testing.T) {
	ReadInString(source)

	require.Equal(t, "Tom Preston-Werner", GetString("owner.name", ""))
	require.Equal(t, "GitHub Cofounder & CEO\nLikes tater tots and beer.", GetString("owner.bio", ""))

	tm, err := time.ParseInLocation("2006-01-02 15:04:05Z", "1979-05-27 07:32:00Z", time.UTC)
	if err != nil {
		panic(err)
	}
	require.Equal(t, tm, GetOffsetDateTime("owner.dob", time.Now()))
}

const source = `
# This is a TOML document Copied from 
#   https://raw.githubusercontent.com/BurntSushi/toml/master/_examples/example.toml

title = "TOML Example"

[owner]
name = "Tom Preston-Werner"
organization = "GitHub\""
bio = "GitHub Cofounder & CEO\nLikes tater tots and beer."
dob = 1979-05-27 07:32:00Z # First class dates? Why not?
dob1 = 1979-05-27

[database]
server = "192.168.1.1"
ports = [ 8001, 8001, 8002 ]
connection_max = 5000
enabled = true

[servers]

  # You can indent as you please. Tabs or spaces. TOML don't care.
  [servers.alpha]
  ip = "10.0.0.1"
  dc = "eqdc10"

  [servers.beta]
  ip = "10.0.0.2"
  dc = "eqdc10"

[clients]
data = [ ["gamma", "delta"], [1, 2] ] # just an update to make sure parsers support it

# Line breaks are OK when inside arrays
hosts = [
  "alpha",
  "omega"
]
`
