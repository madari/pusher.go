Pusher.go - HTTP Push package for Google Go
==========================================

Pusher.go gives existing and new [Google Go](http://golang.org/) applications
real and simple
[HTTP Server Push](http://en.wikipedia.org/wiki/Push_technology#HTTP_server_push)
capabilities without requiring any major efforts. Pusher.go relies and conforms
to the
[Basic HTTP Push Relay Protocol](http://pushmodule.slact.net/protocol.html)
to provide understandable publisher/subscriber mechanism above uniquely
identifiable channels following the footsteps of its' inspirational source,
the [NGiNX HTTP push module](http://pushmodule.slact.net/).

## Documentation

[Full documentation](http://gopkgdoc.appspot.com/pkg/github.com/madari/pusher.go).

## Client

A reference client written in JavaScript is available under the [example](example/www/pusher.js).
It should be fairly straightforward to port it into your favourite environment.

## License 

(The MIT License)

Copyright (c) 2011 Jukka-Pekka Kekkonen &lt;karatepekka@gmail.com&gt;

Permission is hereby granted, free of charge, to any person obtaining
a copy of this software and associated documentation files (the
'Software'), to deal in the Software without restriction, including
without limitation the rights to use, copy, modify, merge, publish,
distribute, sublicense, and/or sell copies of the Software, and to
permit persons to whom the Software is furnished to do so, subject to
the following conditions:

The above copyright notice and this permission notice shall be
included in all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED 'AS IS', WITHOUT WARRANTY OF ANY KIND,
EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY
CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT,
TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE
SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
