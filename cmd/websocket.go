package cmd

// var websocketCmd = &cobra.Command{
// 	Use:   "websocket",
// 	Short: L("Open a websocket connection"),
// 	Long:  L("Open a websocket connection"),
// 	Run: func(cmd *cobra.Command, args []string) {
// 		defer share.SessionStop()
// 		defer plugin.KillAll()
// 		defer func() {
// 			err := exception.Catch(recover())
// 			if err != nil {
// 				fmt.Println(color.RedString(L("Fatal: %s"), err.Error()))
// 			}
// 		}()

// 		Boot()
// 		cfg := config.Conf
// 		cfg.Session.IsCLI = true
// 		engine.Load(cfg)
// 		if len(args) < 1 {
// 			fmt.Println(color.RedString(L("Not enough arguments")))
// 			fmt.Println(color.WhiteString(share.BUILDNAME + " help"))
// 			return
// 		}

// 		name := args[0]
// 		websocket, has := websocket.WebSockets[name]
// 		if !has {
// 			fmt.Println(color.RedString(L("%s not exists!"), name))
// 			return
// 		}

// 		url := websocket.URL
// 		protocols := websocket.Protocols
// 		argsLen := len(args)
// 		if argsLen > 1 {
// 			url = args[1]
// 		}

// 		if argsLen > 2 {
// 			protocols = args[2:]
// 		}

// 		fmt.Println(color.WhiteString("\n---------------------------------"))
// 		fmt.Println(color.WhiteString(websocket.Name))
// 		fmt.Println(color.WhiteString("---------------------------------"))
// 		fmt.Println(color.GreenString("      URL: %s", url))
// 		fmt.Println(color.GreenString("Protocols: %s", strings.Join(protocols, ",")))
// 		fmt.Println(color.WhiteString("--------------------------------------"))
// 		pargs := append([]string{url}, protocols...)
// 		err := websocket.Open(pargs...)
// 		if err != nil {
// 			fmt.Println(color.RedString(L("%s"), err.Error()))
// 			return
// 		}
// 	},
// }
