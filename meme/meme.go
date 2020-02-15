package meme

import (
	"fmt"
	"github.com/ethanzeigler/groupme/gmbots/adapter"
	"github.com/sirupsen/logrus"
	"math/rand"
	"regexp"
	"strconv"
	"strings"

	srv "github.com/ethanzeigler/groupme/botserver"
)

var idMap map[string]string
var quoteRegex *regexp.Regexp

func init() {
	quoteRegex = regexp.MustCompile(`^(?i)/(?P<Name>.+)ism(?:\s+(?P<Subcommand>record|delete)\s*(?P<Argument>.+)?|(?P<ImproperData>.*))?\s*$`)
}

// Create the meme machine channel
func MakeMemeChannel(db *adapter.MemeDB) (channel srv.Channel) {
	c := &channel
	c.Name = "Meme Machine"
	// Stores the group IDs this channel will listen to
	c.GroupIDs = []string{"01234", "46818924"}
	idMap = map[string]string{
		// TSCE
		"30154628": "6df3e3f76b0bfd181f4c41f343",
		// testing
		"46818924": "92c14c082f6e0f860072542b7c",
	}

	// Create Hooks

	// Create hook responsible for the quote system, managing the
	// quote database and other functions
	quoteHook := srv.BasicHook{DebugName: "Quote System", Handler: quoteRequest}
	c.AddHook(&quoteHook)

	roastedHook := srv.BasicHook{DebugName: "Roasted", Handler: roasted}
	c.AddHook(&roastedHook)

	connectFourHook := srv.BasicHook{DebugName: "Connect 4", Handler: connectFour}
	c.AddHook(&connectFourHook)

	helpHook := srv.BasicHook{DebugName: "Help", Handler: helpCommand}
	c.AddHook(&helpHook)

	pikachuHook := srv.BasicHook{DebugName: "Pikachu", Handler: pikachu}
	c.AddHook(&pikachuHook)

	justRightHook := srv.BasicHook{DebugName: "Just right", Handler: justRight}
	c.AddHook(&justRightHook)
	return
}

func quoteRequest(callback srv.Callback, i *srv.Instance) (cont bool) {
	// check if responsible
	matches := quoteRegex.FindStringSubmatch(callback.Text)

	// no match?
	if matches == nil {
		cont = false
		return
	}

	names := quoteRegex.SubexpNames()
	captureGroups := mapSubexpNames(matches, names)


	i.Log.WithFields(logrus.Fields{
		"matches": matches,
		"captures": captureGroups,
	}).Debug("Recognized quote request")

	// This will be handed successfully, so we can take ownership
	cont = true
	msg := srv.Message{BotID: idMap[callback.GroupID]}

	// is the command used correctly?
	if hasGroup(captureGroups, "ImproperData") ||
		(hasGroup(captureGroups, "Subcommand") && !hasGroup(captureGroups, "Argument")) {
		msg.Text = "Hmm. I don't understand this extra information. Did you want a subcommand? (/commands)"
		i.PostMessageAsync(msg, 2)
		return
	}

	// Get quotee name
	selectedName := strings.TrimSpace(captureGroups["Name"])

	// subcommand and argument?
	if hasGroup(captureGroups, "Subcommand") && hasGroup(captureGroups, "Argument") {
		subcommand := strings.TrimSpace(captureGroups["Subcommand"])
		argument := strings.TrimSpace(captureGroups["Argument"])

		// =================================
		// We need to check for a sub command
		// Wow this code is getting big and messy
		// =================================

		// record subcommand
		if strings.EqualFold(subcommand, "record") {
			i.Log.("Recording Quote " + selectedName)

			// Write quote to the psql db
			err := adapter.WriteUserQuote(selectedName, argument, callback)
			if err == nil {
				i.LogDebug("Success!")
				msg.Text = "👍"
				i.PostMessageAsync(msg, 2)
			} else {
				i.LogError("Couldn't record: " + err.Error())
				msg.Text = "[Error: Reported to developer] " + err.Error()
				i.PostMessageAsync(msg, 2)
			}
		}

		// delete subcommand
	} else if hasGroup(captureGroups, "Subcommand") {
		subcommand := strings.TrimSpace(captureGroups["Subcommand"])

		if strings.EqualFold(subcommand, "delete") {
			srv.LogDebug("Deleting quote")
			quote, err := adapter.GetQuotes(selectedName, callback, 1, adapter.QuoteIDSort)
			if err != nil {
				if err.Error() == "no quotes found" {
					srv.LogDebug("Quote doesn't exist")
					msg.Text = "Can't delete quote: " + err.Error()
					srv.PostMessageAsync(msg, 2)
				} else {
					srv.LogDebug("Failed to connect to db")
					msg.Text = "[Error: Reported to developer] " + err.Error()
					srv.PostMessageAsync(msg, 2)
				}
			} else {
				// check that it's sent by the person who originally submitted the quote
				if *quote[0].SubmitterID == callback.SenderID {
					_, err := adapter.DeleteQuote(quote[0])
					if err != nil {
						msg.Text = "Couldn't delete that quote: " + err.Error()
						srv.PostMessageAsync(msg, 2)
					} else {
						msg.Text = fmt.Sprintf("Deleted '%s'", *quote[0].Quote)
						srv.PostMessageAsync(msg, 2)
					}
				} else {
					// someone else is trying to delete the quote
					msg.Text = "Only the person who wrote the quote can delete it"
					srv.PostMessageAsync(msg, 2)
				}
			}
		} else {
			srv.LogWarning("Bad input interpreted as a subcommand")
			msg.Text = "Internal error. Misinterpreted the message."
			srv.PostMessageAsync(msg, 2)
		}
	} else {
		// there isn't a subcommand. Get a quote from the person

		quote, err := adapter.GetUserQuote(selectedName, callback)
		if err != nil {
			srv.Log.WithFields(logrus.Fields{
				"err":   err.Error(),
				"name":  selectedName,
				"group": callback.GroupID,
			}).Error("Cannot query database")
			msg.Text = fmt.Sprint("Cannot get quote: " + err.Error())
			srv.PostMessageAsync(msg, 2)
			return
		}
		// write first letter of name
		msg.Text += strings.ToUpper(string(selectedName[0]))
		// write rest of name and quote
		date := quote.Date.Format("Mon, Jan 2, 1970")
		msg.Text += fmt.Sprintf("%s [%s]: %s", selectedName[1:], date, *quote.Quote)
		s.PostMessageAsync(msg, 2)
	}
	return
}

func roasted(callback srv.Callback, i *srv.Instance) (cont bool) {
	if strings.EqualFold(callback.Text, "/roasted") {
		msg := srv.Message{BotID: idMap[callback.GroupID]}
		msg.Picture = "https://i.groupme.com/750x703.jpeg.4bc7c92a3a23460da1dff0c2490de22f"
		s.PostMessageAsync(msg, 2)
		cont = true
	} else {
		cont = false
	}
	return
}

func connectFour(callback srv.Callback, i *srv.Instance) (cont bool) {
	matched, _ := regexp.Match("^(?i)(.*\\s)?/c4(\\s[1-9])?$", []byte(callback.Text))
	if matched {
		cont = true
		msg := srv.Message{BotID: idMap[callback.GroupID]}
		args := strings.Split(strings.TrimSpace(callback.Text), " ")
		if len(args) > 1 {
			count, _ := strconv.Atoi(args[1])
			defer func(n int) {
				for i := 0; i < count; i++ {
					msg.Picture = c4Images[rand.Intn(len(c4Images))]
					srv.PostMessageSync(msg, 1)
				}
			}(count)
		} else {
			msg.Picture = c4Images[rand.Intn(len(c4Images))]
			srv.PostMessageAsync(msg, 2)
		}
	} else {
		cont = false
	}
	return
}

func pikachu(callback srv.Callback, i *srv.Instance) (cont bool) {
	matches, _ := regexp.Match("^(?i)(.*\\s)?/pika$", []byte(callback.Text))
	if matches {
		cont = true
		msg := srv.Message{BotID: idMap[callback.GroupID]}
		msg.Picture = "https://i.groupme.com/1354x784.png.75b2bbb3210c463094551c5dbf396672"
		srv.PostMessageAsync(msg, 2)
	} else {
		cont = false
	}
	return
}

func justRight(callback srv.Callback, s *srv.Instance) (cont bool) {
	matches, _ := regexp.Match("^(?i)(.*\\s)?/just\\sright$", []byte(callback.Text))
	if matches {
		cont = true
		msg := srv.Message{BotID: idMap[callback.GroupID]}
		msg.Picture = "https://i.groupme.com/480x480.jpeg.f880c37db898434fbe7def6504225c7d"
		srv.PostMessageAsync(msg, 2)
	} else {
		cont = false
	}
	return
}

func helpCommand(callback srv.Callback, i *srv.Instance) (cont bool) {
	matches, _ := regexp.Match("^(?i)/help$", []byte(callback.Text))
	if matches {
		cont = true
		msg := srv.Message{BotID: idMap[callback.GroupID]}
		msg.Text = "/<name>ism [record <message>] - Group member quotes and adding new ones\n" +
			"/just right - Hercules meme\n" +
			"/c4 [1-9] - Connect 4 memes\n" +
			"/pika - Pikachu surprised meme\n" +
			"/roasted - Roasted by the group meme\n"
		srv.PostMessageAsync(msg, 2)
	} else {
		cont = false
	}
	return
}

//////////////////////////////////////////////////////////////////////////////////
//////////////////////////////////////////////////////////////////////////////////
//////////////////////////////////////////////////////////////////////////////////

//////////////////////////////////////////////////////////////////////////////////
//////////////////////////////////////////////////////////////////////////////////
//////////////////////////////////////////////////////////////////////////////////

//////////////////////////////////////////////////////////////////////////////////
//////////////////////////////////////////////////////////////////////////////////
//////////////////////////////////////////////////////////////////////////////////

var c4Images = [...]string{"https://i.groupme.com/500x496.gif.61a5622e82cb4b01835f8b5e241de5cf",
	"https://i.groupme.com/500x499.jpeg.7510ef839181406da2786584ef7fcfec",
	"https://i.groupme.com/500x505.png.f4a0d2c9cf134b21801904d62b1bd7c2",
	"https://i.groupme.com/500x503.jpeg.40a8c439214e4303a9c42918841fde34",
	"https://i.groupme.com/750x729.jpeg.051dfb09a3394eb49ca2471cab4158aa",
	"https://i.groupme.com/750x600.jpeg.b7b960f7bd294f4da993688dae3d35b8",
	"https://i.groupme.com/750x720.jpeg.052fcbc7df53496e8a0ab0525babbf92",
	"https://i.groupme.com/461x450.jpeg.3bbd1c8a635344ba9e46eb6b7ea42417",
	"https://i.groupme.com/400x411.jpeg.7eaec93b26004cdfa781d704badefcf7",
	"https://i.groupme.com/400x403.jpeg.bb7b8d0a571049388afb70b8b269c86a",
	"https://i.groupme.com/400x357.jpeg.6cb3569bf7d146339a0afc85541d4be3",
	"https://i.groupme.com/400x401.jpeg.4452223fc0784e94bf4c158c938f361e",
	"https://i.groupme.com/400x482.jpeg.6308c8ee89154efcb5d9ad3c955031fe",
	"https://i.groupme.com/500x464.jpeg.9eee3dec2fbc4906ad7c41f3e551226a",
	"https://i.groupme.com/500x499.jpeg.5444871acc614df1bd2d62577b32243c",
	"https://i.groupme.com/500x482.jpeg.bf69575c06044816940c4a38cc0a0bd4",
	"https://i.groupme.com/500x497.jpeg.c2ba665554c64b33a4153cbd66e87a64",
	"https://i.groupme.com/500x499.jpeg.c5fed973a2e14300b450747ed7a58764",
	"https://i.groupme.com/500x499.jpeg.fc5adf89ed284c9c9f116471b2e4c607",
	"https://i.groupme.com/640x640.jpeg.0995b94db2a14d2e8126ced90e198df3",
	"https://i.groupme.com/640x608.jpeg.24ace29d45b14e4688b1d04ab772468f",
	"https://i.groupme.com/640x642.jpeg.f5dae0718c294e7a8904a2a609e52768",
	"https://i.groupme.com/750x741.jpeg.43ed251e951847d79ab754d35cdf5744",
	"https://i.groupme.com/750x713.jpeg.1e2074cfee884d7f8600d20f64ffa21c",
	"https://i.groupme.com/400x359.jpeg.a088cefe7d404d94a6fd01f02c672f61",
	"https://i.groupme.com/500x485.jpeg.3abd0e64892c459fb80696f4f1563434",
	"https://i.groupme.com/458x462.jpeg.7776b4bdf8084beea0556a8134077370",
	"https://i.groupme.com/500x567.jpeg.ec9c9abcb00748008533e55fed3d3e97",
	"https://i.groupme.com/640x587.jpeg.27cae6e085cd44228eb160d22c754e06",
	"https://i.groupme.com/750x740.jpeg.749050b93d174d78a411928527d0ae60",
	"https://i.groupme.com/750x754.jpeg.ac3c1af4f97446959765b6cdba995297",
	"https://i.groupme.com/640x781.jpeg.fd6ba466dd114e6188521ba2eca44c0e",
	"https://i.groupme.com/640x640.jpeg.7e713cc6d7a1437e8ac55b9c95ab20e4",
	"https://i.groupme.com/500x500.jpeg.83418d5abe914d0ea70e2c197e1b31a4",
	"https://i.groupme.com/2160x2168.jpeg.4946eecc301d45d6b75c6978ed73bdef",
	"https://i.groupme.com/640x635.jpeg.c9be5a78a6a4422db2fdeb1ef969e990",
	"https://i.groupme.com/400x399.jpeg.eba2af45c3f94f1aa4c573ac873c6f02",
	"https://i.groupme.com/680x671.jpeg.d42f9d9946a540b7b82358842a26f629",
	"https://i.groupme.com/400x398.jpeg.99813a93045d45a2a5402e80d481f087",
	"https://i.groupme.com/500x490.jpeg.592f8ebb24cb45c1a55a7bdf6c7138da",
	"https://i.groupme.com/500x508.jpeg.975ddf42fbcb422f9d4c69e7fa8eb18a",
	"https://i.groupme.com/400x386.jpeg.379178be4a4447ad8f96cef7209fffe0",
	"https://i.groupme.com/680x676.jpeg.d0b58e38f2314c3cbbbf2977fb256e29",
	"https://i.groupme.com/640x640.jpeg.c1389a8adcb64999b1fc2f83cbecdc06",
	"https://i.groupme.com/750x729.jpeg.26999b07d83c4720ae452dc4f119c338",
	"https://i.groupme.com/750x743.jpeg.69c89268ca9747ac93d66c93973f0038",
	"https://i.groupme.com/750x731.jpeg.27fc3373091b4254bfa63ce5d04b4005",
	"https://i.groupme.com/748x838.jpeg.7637f1e90c9741538497697d3e7d9887",
	"https://i.groupme.com/749x859.jpeg.cacafd04c7984837bcc66be0c6adee73",
	"https://i.groupme.com/750x763.jpeg.811217cfecbd4ec1b3e953f18a17abf9",
	"https://i.groupme.com/500x497.jpeg.97395c0ca5e4490f877cac7de2de4f5f",
	"https://i.groupme.com/750x746.jpeg.b4f9574493a34d4aa6c30a0fd37c5b06",
	"https://i.groupme.com/742x974.jpeg.0248c2f4b07b4263b357e81fb5567098",
	"https://i.groupme.com/634x630.jpeg.7c40bab49a3d4bc8b78cab243d70f4c0",
	"https://i.groupme.com/680x676.jpeg.df98e2dd6fa74584a1613b52ac081dbe",
	"https://i.groupme.com/750x743.jpeg.bf99334eb75348aa9db1e26de6cd11b0",
	"https://i.groupme.com/661x675.jpeg.a41dfb168a234c0088254735c65c1498",
	"https://i.groupme.com/400x361.jpeg.93e565f5e6924c189bbccc40beb86c84",
	"https://i.groupme.com/375x384.jpeg.3f2403cf36844e2980eba5cb2bd3bc9f",
	"https://i.groupme.com/750x747.jpeg.b5f2345e72d44ccf8476f6302f95dd52",
	"https://i.groupme.com/750x738.jpeg.f2adbb06284d44cdba1e2b76b1f98497",
	"https://i.groupme.com/500x518.png.75139f1e022b42c7a1375479ba507966",
	"https://i.groupme.com/371x351.jpeg.e07f81b61d61401d97538f2834a511f7",
	"https://i.groupme.com/366x350.jpeg.381e0bff5c494e9988d7939a83fdc0b8",
	"https://i.groupme.com/224x225.jpeg.20aff0124fdc4283815b93e087b063ec",
	"https://i.groupme.com/230x219.jpeg.2b7980ed1b3d4dbabb9f4f7b0129c877",
	"https://i.groupme.com/225x224.jpeg.9da55929872b498c9af1db99e1b460ae",
	"https://i.groupme.com/226x223.jpeg.1b4704ecc2eb490f845dfd8cdb30d387",
	"https://i.groupme.com/225x225.jpeg.f0e70521a3dd40f09fbea799a8bb2d7c",
	"https://i.groupme.com/220x229.jpeg.794838b7f09541c19a846ce38c74aa1f",
	"https://i.groupme.com/314x161.jpeg.c14eefda9a124705850a9e381fc58137",
	"https://i.groupme.com/225x225.jpeg.7bdc0e40da7d4baa8aa8fd85fc10d129",
	"https://i.groupme.com/680x676.png.5c0beb4f78144414983adfa5ccd5d8d4",
	"https://i.groupme.com/749x857.jpeg.2eea6484d45e448dbc65b720b95eeb99",
	"https://i.groupme.com/634x543.jpeg.1f50a6e6ab0541719852dca47c3ebf93",
	"https://i.groupme.com/1080x1476.png.6088d73e539a4c62a5723117479fdfbc",
	"https://i.groupme.com/583x565.jpeg.7964796f5f224fa3998dce23bd73bd24",
	"https://i.groupme.com/680x676.jpeg.0ed178000aad480eb108f18cd858686a",
	"https://i.groupme.com/232x217.jpeg.270f4233c00d4056a770f2dfb6f22bc4",
	"https://i.groupme.com/640x640.jpeg.78280821ef964e94b28b5560afa85977",
	"https://i.groupme.com/602x590.jpeg.5c87b4315a474aafbc77b1b4daac770e",
	"https://i.groupme.com/634x630.png.8de15b7d913541ec8e036ede709cd15f",
	"https://i.groupme.com/600x600.jpeg.8beba1fc62ab480f9e78627884a4785e",
	"https://i.groupme.com/500x499.png.10db9437e08643239c67622bf10eeec6",
	"https://i.groupme.com/609x602.jpeg.2daf2ef985d54fd6a47f876025f4ec71",
	"https://i.groupme.com/500x519.png.03a8a69e17c044fb9d50e653c0a72377",
	"https://i.groupme.com/236x234.jpeg.fd11b5eaea654445a9bdf3915f7af66b",
	"https://i.groupme.com/634x630.png.b568f4928eb0488aa21e6ea8c394d359",
	"https://i.groupme.com/300x289.jpeg.d1fbdd2cded5404497edc6eff1a0ed2f",
	"https://i.groupme.com/320x320.jpeg.8be400326a6e43a788908e8331de86be",
	"https://i.groupme.com/480x480.jpeg.b180819b6c1c46aa97a3c05f0c912a6a",
	"https://i.groupme.com/640x591.jpeg.f3a3b5c4eb7e4f27a3a5c8ecc9045402",
	"https://i.groupme.com/634x630.jpeg.a0cbd8454f5f4e05b1d2e0b904325866",
	"https://i.groupme.com/506x498.jpeg.f2e900923e654acd97138b642eba79a3",
	"https://i.groupme.com/768x676.jpeg.4b6bf4a26c4e4e22b995f6bfb04c5b1d",
	"https://i.groupme.com/501x497.jpeg.ba02363d9f78452e976891446ecd67ba",
	"https://i.groupme.com/532x526.jpeg.11e449ce364a48dbb34d4380e8d509cd",
	"https://i.groupme.com/525x489.jpeg.4e94827f766049d0b4761f76d43a0142"}

// StackOverflow @thwd
// Map capture group names to values
// Because this isn't native for some reason...
func mapSubexpNames(m, n []string) map[string]string {
	m, n = m[1:], n[1:]
	r := make(map[string]string, len(m))
	for i := range n {
		r[n[i]] = m[i]
	}
	return r
}

// Returns true if the given map contains the given key
func hasGroup(m map[string]string, key string) bool {
	if val, ok := m[key]; ok {
		return len(strings.TrimSpace(val)) > 0
	}
	return false
}
