# WebSocket
Go Web Socket implementation as defined in protocol RFC 6455.

//TODO:HTTP VERSION FROM RESPONSE SAME AS REQUEST
//TODO: HEAD REQUESTS


//TODO:Implement chunked transfer encoding if dealing with large responses.

//TODO:Content Serving:
    Serve static files from the filesystem.
    Implement a basic way to handle dynamic content, perhaps through a simple CGI mechanism or templating.

//TODO:Concurrency:
    Use pool of threads to handle requests instead of thread per connection.
//TODO:Logging:
    Add logging of requests and responses for debugging and analytics.
//TODO:Error Handling and Testing:
    Implement robust error handling.
    Write tests to ensure your server responds correctly to various HTTP requests.
    

//TODO:Security Enhancements:
    Consider HTTPS support (this would involve setting up SSL/TLS).
//TODO:Performance Tuning:
    Optimize for speed
