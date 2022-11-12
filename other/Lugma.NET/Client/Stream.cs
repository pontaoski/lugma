namespace Lugma.Client;

public interface IStream<TExtra>
{
    void Unsubscribe(int handlerNumber);
    int Subscribe<TRecv>(string evt, Action<TRecv> callback);
    int SubscribeToClose(Action callback);
    Task Send<TSend>(string signal, TSend body);
}
