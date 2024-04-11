import { parse } from "path";
import { decode, encode, print } from "./codec.js";
import { Socket } from 'net';

export class WebSocketAgent {
    constructor() {
        this._socket = null;
    }
    connect(host, port) {
        this._socket = new WebSocket(`ws://${host}:${port}`);
        this._socket.addEventListener('message', (event) => {
            print(decode(event.data));
        })
        this._socket.addEventListener("close", (event) => {
            print("connect close...")
        })
        this._timer = setInterval(() => {
            const ping = Buffer.alloc(1)
            ping[0] = 0x9
            this._socket.send(ping)
        })
    }

    send(data) {
        this._socket.send(data);
    }

    on(event, callback) {
        this._socket.addEventListener(event, callback);
    }

    close() {
        this._socket.close()
        clearInterval(this._timer)
    }
}

const pkgSizeByteLen = 4

export class TCPAgent {
    _buf = Buffer.alloc(0)
    constructor() {
        this._socket = null
        this._buf = Buffer.alloc(0)
        Buffer.concat([this._buf, this._buf])

    }

    ping() {
        const ping = Buffer.alloc(pkgSizeByteLen)
        ping.writeUint32LE(0)
        this._socket.write(ping)
    }

    connect(host, port) {
        this._socket = new Socket()
        this._socket.connect(port, host, () => {
            print("connect success")
            this._timer = setInterval(() => {
                this.ping()
            }, 3000)
        })

        this._socket.on("data", (data) => {
            this._buf = Buffer.concat([this._buf, data])
            if (this._buf.length < pkgSizeByteLen) {
                return
            }
            let pkgSize = this._buf.readUInt32LE()
            let sliceLen = pkgSize + pkgSizeByteLen
            if (this._buf.length < sliceLen) {
                return
            }
            let pkg = this._buf.subarray(pkgSizeByteLen, sliceLen)
            this._buf = this._buf.subarray(sliceLen)
            try {
                let [msgName, msg] = decode(pkg)
                print("recv server msg:", msgName, msg)
            } catch {
                print("decode error", data, data.length)
            }
        })
    }

    send(msgName = 'SayHelloReq', msgBody = { text: "hello, server!" }) {
        let b = encode(msgName, msgBody)
        this._socket.write(b)
    }

    close() {
        this._socket.destroy()
        clearInterval(this._timer)
    }
}