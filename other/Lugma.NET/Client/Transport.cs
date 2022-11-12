namespace Lugma.Client;

public interface ITransport<TExtra>
{
    Task<TReturn> MakeRequest<TRequest, TReturn, TError>(string endpoint, TRequest body, TExtra? extra);
    Task<IStream<TExtra>> OpenStream(string endpoint, TExtra? extra);
}
