namespace Lugma.Client.HTTP;

using System.Net;
using System.Net.Http.Headers;
using System.Net.Http.Json;
using System.Threading.Tasks;

public class HTTPSTransport : ITransport
{
    private Uri BaseURI;
    private HttpClient Client;
    public HTTPSTransport(Uri baseURI)
    {
        BaseURI = baseURI;
        Client = new HttpClient();
    }
    public async Task<TReturn> MakeRequest<TRequest, TReturn, TError>(
        string endpoint, TRequest body, Metadata metadata)
    {
        var path = new Uri(BaseURI, endpoint);

        var content = JsonContent.Create(body);
        content.Headers.Add("Content-Type", "application/json");
        foreach (var item in metadata)
            content.Headers.Add($"lugma-{item.Key}", item.Value);

        var response = await Client.PostAsync(path, content);
        if (response.StatusCode == HttpStatusCode.OK)
        {
            return (await response.Content.ReadFromJsonAsync<TReturn>())!;
        }
        else
        {
            var err = await response.Content.ReadFromJsonAsync<TError>();
            throw new RPCError<TError>(err!);
        }
    }
    public async Task<IStream> OpenStream(string endpoint, Metadata extra)
    {
        var path = new Uri(BaseURI, endpoint);
    }
}
