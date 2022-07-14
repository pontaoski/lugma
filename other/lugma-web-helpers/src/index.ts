export interface Transport<T> {
    makeRequest(endpoint: string, body: any, extra: T | undefined): Promise<any>
    openStream(endpoint: string, extra: T | undefined): Stream
}
export interface Stream {
    unon(item: number): void
    on(event: string, callback: (body: any) => void): number
    onClose(callback: () => void): number
    send(signal: string, body: any): void
}

export class WebSocketStream implements Stream {
    socket: WebSocket
    callbacks: Map<number, (body: any) => void>
    eventsToNumbers: Map<string, Set<number>>
    numbersToEvents: Map<number, string>
    callbackNumber: number

    constructor(url: URL, initialPayload: any) {
        this.socket = new WebSocket(url)
        this.socket.binaryType = "arraybuffer"
        this.callbacks = new Map()
        this.eventsToNumbers = new Map()
        this.numbersToEvents = new Map()
        this.callbackNumber = 0

        this.socket.addEventListener("open", () => {
            this.socket.send(JSON.stringify(initialPayload))
        })
        this.socket.addEventListener("close", () => {
            const values = this.eventsToNumbers.get("on closed")?.values()
            if (values == undefined) {
                return
            }
            for (let value of values) {
                this.callbacks.get(value)?.(undefined)
            }
        })
        this.socket.addEventListener("message", (msg) => {
            if (msg.data instanceof ArrayBuffer) {

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
    send(signal: string, body: any): void {
        this.socket.send(JSON.stringify({
            "type": signal,
            "content": body,
        }))
    }
}
export class HTTPSTransport implements Transport<Headers> {
    baseURL: URL

    constructor(baseURL: URL) {
        this.baseURL = baseURL
    }
    async makeRequest(endpoint: string, body: any, extra: Headers | undefined = undefined): Promise<any> {
        const path = new URL(endpoint, this.baseURL)
        const headers: {[key: string]: string} = {
            'Content-Type': 'application/json'
        }
        extra?.forEach((val, key) => {
            headers[key] = val
        })
        const response = await fetch(path.toString(), {
            method: 'POST',
            body: JSON.stringify(body),
            headers: headers
        })
        const json = await response.json()
        if (response.status === 200) {
            return json
        } else {
            throw json
        }
    }
    openStream(endpoint: string, extra: Headers | undefined): Stream {
        const path = new URL(endpoint, this.baseURL)

        const headers: {[key: string]: string} = {}
        extra?.forEach((val, key) => {
            headers[key] = val
        })

        return new WebSocketStream(path, headers)
    }
}
