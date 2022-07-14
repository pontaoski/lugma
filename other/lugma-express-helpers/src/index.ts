import { Stream, Transport, Result } from "lugma-server-helpers"
import * as Express from "express"
import * as HTTP from "http"
import * as WebSockets from "ws"
import * as ExpressWebsockets from "express-ws"

export class WebSocketStream implements Stream<HTTP.IncomingHttpHeaders> {
    socket: WebSockets.WebSocket
    callbacks: Map<number, (body: any) => void>
    eventsToNumbers: Map<string, Set<number>>
    numbersToEvents: Map<number, string>
    callbackNumber: number

    constructor(socket: WebSockets.WebSocket) {
        this.socket = socket
        this.socket.binaryType = "arraybuffer"
        this.callbacks = new Map()
        this.eventsToNumbers = new Map()
        this.numbersToEvents = new Map()
        this.callbackNumber = 0

        this.socket.addEventListener("close", () => {
            const values = this.eventsToNumbers.get("on closed")?.values()
            if (values == undefined) {
                return
            }
            for (let value of values) {
                this.callbacks.get(value)?.(undefined)
            }
        })
        let initialPayload: any | null = null
        this.socket.addEventListener("message", (msg) => {
            if (typeof msg.data === "string") {
                if (initialPayload === null) {
                    initialPayload = JSON.parse(msg.data)

                    const values = this.eventsToNumbers.get("on initial")?.values()
                    if (values == undefined) {
                        return
                    }
                    for (let value of values) {
                        this.callbacks.get(value)?.(initialPayload)
                    }
                } else {
                    const item = JSON.parse(msg.data)
    
                    const values = this.eventsToNumbers.get(item["kind"])?.values()
                    if (values == undefined) {
                        return
                    }
                    for (let value of values) {
                        this.callbacks.get(value)?.(item["content"])
                    }
                }
            }
        })
    }
    unon(item: number): void {
        this.callbacks.delete(item)
        const event = this.numbersToEvents.get(item)
        if (event == undefined) {
            return
        }
        this.numbersToEvents.delete(item)

        const set = this.eventsToNumbers.get(event)
        set?.delete(item)
    }
    on(event: string, callback: (body: any) => void): number {
        this.callbackNumber++
        this.callbacks.set(this.callbackNumber, callback)

        const set = this.eventsToNumbers.get(event) ?? new Set()
        set.add(this.callbackNumber)
        this.eventsToNumbers.set(event, set)
        this.numbersToEvents.set(this.callbackNumber, event)

        return this.callbackNumber
    }
    onClose(callback: () => void): number {
        this.callbackNumber++
        this.callbacks.set(this.callbackNumber, callback)

        const set = this.eventsToNumbers.get("on closed") ?? new Set()
        set.add(this.callbackNumber)
        this.eventsToNumbers.set("on closed", set)
        this.numbersToEvents.set(this.callbackNumber, "on closed")

        return this.callbackNumber
    }
    onOpen(callback: (initialPayload: any) => void): number {
        this.callbackNumber++
        this.callbacks.set(this.callbackNumber, callback)

        const set = this.eventsToNumbers.get("on initial") ?? new Set()
        set.add(this.callbackNumber)
        this.eventsToNumbers.set("on initial", set)
        this.numbersToEvents.set(this.callbackNumber, "on initial")

        return this.callbackNumber
    }
    send(event: string, body: any): void {
        this.socket.send(JSON.stringify({
            "type": event,
            "content": body,
        }))
    }
}

export class ExpressTransport implements Transport<HTTP.IncomingHttpHeaders> {
    router: Express.Application
    websocket: ExpressWebsockets.Instance

    constructor() {
        this.router = Express.default()
        this.router.use(Express.json())
        this.websocket = ExpressWebsockets.default(this.router, undefined, { leaveRouterUntouched: true })
    }
    bindMethod(path: string, slot: (content: any, extra: any) => Promise<Result<any, any>>): void {
        this.router.post(path, async (request, response) => {
            const [kind, ret] = await slot(request.body, request.headers)

            if (kind == "error") {
                response.status(400).json(ret)
            } else {
                response.json(ret)
            }
        })
    }
    bindStream(path: string, slot: (stream: Stream<HTTP.IncomingHttpHeaders>) => void): void {
        this.websocket.app.ws(path, (ws) => {
            slot(new WebSocketStream(ws))
        })
    }
}
