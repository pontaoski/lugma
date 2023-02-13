namespace Lugma.Client.HTTP;

using System.Net.WebSockets;
using System.Net.Http.Headers;
using System.Threading.Tasks;
using System.Text.Json;
using System;
using Websocket.Client;

public class WebSocketStream : IStream
{
    private WebsocketClient WebSocket;
    private Dictionary<int, Action<object>> Callbacks;
    private Dictionary<string, HashSet<int>> EventsToNumbers;
    private Dictionary<int, string> NumbersToEvents;
    private int CallbackNumber;

    internal WebSocketStream(Uri url, object initialPayload)
    {
        WebSocket = new(url);
        WebSocket.IsReconnectionEnabled = false;

        Callbacks = new();
        EventsToNumbers = new();
        NumbersToEvents = new();
        CallbackNumber = 0;

        WebSocket.Start();
        WebSocket.Send(JsonSerializer.Serialize(initialPayload));
        WebSocket.MessageReceived.Subscribe(OnMessage);
        WebSocket.DisconnectionHappened.Subscribe(OnClose);
    }
    private void OnMessage(ResponseMessage msg)
    {
        if (msg.MessageType != WebSocketMessageType.Text)
            return;
    }
    private void OnClose(DisconnectionInfo info)
    {

    }

    public void Dispose()
    {
        WebSocket.Dispose();
    }

    public Task Send<TSend>(string signal, TSend body)
    {

        throw new NotImplementedException();
    }

    public int Subscribe<TRecv>(string evt, Action<TRecv> callback)
    {
        throw new NotImplementedException();
    }

    public int SubscribeToClose(Action callback)
    {
        throw new NotImplementedException();
    }

    public void Unsubscribe(int handlerNumber)
    {
        throw new NotImplementedException();
    }
}
