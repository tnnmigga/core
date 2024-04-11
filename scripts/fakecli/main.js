import repl from 'repl'
import { initMsgBuilder } from './codec.js'
import { TCPAgent, WebSocketAgent } from './agent.js'

var address = "127.0.0.1"
var port = 10078
var isTcp = true

initMsgBuilder("pb")

var agent = new TCPAgent()

function connect(addr = address, host = port) {
    if (isTcp) {
        agent = new TCPAgent()
    } else {
        agent = new WebSocketAgent()
    }
    agent.connect(addr, host)
    send("SayHelloReq", { text: "hello, server!" })
}


function runCli(context = {}, name = 'REPL') {
    const r = repl.start({
        // prompt: `${name} > `,
        preview: true,
        terminal: true,
    });
    Object.setPrototypeOf(r.context, context);
    global.console = r.context.console;
}

connect()
runCli({ send, connect })

function send(msgName = 'SayHelloReq', msgBody = { text: "hello, server!" }) {
    agent.send(msgName, msgBody)
}
