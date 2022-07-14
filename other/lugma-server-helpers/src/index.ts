export interface Transport<T> {
    bindMethod(path: string, slot: (content: any, extra: T) => Promise<Result<any, any>>): void
    bindStream(path: string, slot: (stream: Stream<T>) => void): void
}

export interface Stream<T> {
    unon(item: number): void
    on(signal: string, callback: (body: any) => void): number
    onOpen(callback: (initialPayload: T) => void): number
    onClose(callback: () => void): number
    send(event: string, body: any): void
}

export type Result<T, Err> = ["ok", T] | ["error", Err]
