import { Transport, Stream } from 'lugma-web-helpers'
export interface Message {
	id: string
	optionalField: (string|null|undefined)
}
export type Error =
	"Gay" |
	{ NoPermissions: {needed: string;} }
export type Permissions = string
export type Overrides = string
export interface ChatRequests<T> {
	SubscribeToEvents(): ChatStream
	SendMessage(message: Message, homosexuality: Message, extra: T): Promise<void>
}
export interface ChatStream extends Stream {
	onMessageReceived(callback: (message: Message) => void): number
}
export function makeChatFromTransport<T>(transport: Transport<T>): ChatRequests<T> {
	return {
		async SendMessage(message: Message, homosexuality: Message, extra: T): Promise<void> {
			return await transport.makeRequest(
				"Example.lugma/Chat/SendMessage",
				{
					message: message,
					homosexuality: homosexuality,
				},
				extra,
			)
		},
		SubscribeToEvents(): ChatStream {
			return Object.create(
				transport.openStream("Example.lugma/Chat"),
				{
					onMessageReceived: {
						value: function(callback: (message: Message) => void): number {
							return this.on("MessageReceived", callback)
						}
					}
				}
			)
		}
	}
}
