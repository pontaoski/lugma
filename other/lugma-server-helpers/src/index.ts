export interface Metadata {
    [key: string]: string
}

export interface Transport {
    bindMethod(path: string, slot: (content: any, metadata: Metadata) => Promise<Result<any, any>>): void
    bindStream(path: string, slot: (stream: Stream) => void): void
}

export interface Stream {
    unon(item: number): void
    on(signal: string, callback: (body: any) => void): number
    onOpen(callback: (metadata: Metadata) => void): number
    onClose(callback: () => void): number
    send(event: string, body: any): void
}

export type Result<T, Err> = ["ok", T] | ["error", Err]
