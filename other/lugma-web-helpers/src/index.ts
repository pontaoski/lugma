export interface Transport<T> {
    makeRequest(endpoint: string, body: any, extra: T): Promise<any>
    openStream(): Stream
}
export interface Stream {
    unon(item: number): void
    on(event: string, callback: (body: any) => void): number
    onClose(callback: (error: boolean) => void): void
}
