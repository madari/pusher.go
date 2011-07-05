/**
 * @license (The MIT License)
 * 
 * Copyright (c) 2011 Jukka-Pekka Kekkonen <karatepekka@gmail.com>
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy of
 * this software and associated documentation files (the 'Software'), to deal in
 * the Software without restriction, including without limitation the rights to
 * use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
 * the Software, and to permit persons to whom the Software is furnished to do so,
 * subject to the following conditions: The above copyright notice and this
 * permission notice shall be included in all copies or substantial portions of the
 * Software. THE SOFTWARE IS PROVIDED 'AS IS', WITHOUT WARRANTY OF ANY KIND,
 * EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
 * MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO
 * EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR
 * OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING
 * FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
 * THE SOFTWARE.
 */

/**
 * Pusher provides an interface to the Basic HTTP Push Relay Protocol as a part of
 * the Pusher.go HTTP Push package for Google Go.
 *
 * @see http://github.com/madari/pusher.go
 * @see http://pushmodule.slact.net/protocol.html
 *
 * @param {String} subLocation An URL to a subscriber location (i.e. channel)
 * @param {String} [pubLocation] An URL to a publisher location
 */
function Pusher(subLocation, pubLocation) {
	var etag = 0, since = 0;

	/**
	 * URL to a publisher location.
	 *
	 * @type String
	 */
	this.subLocation = subLocation;

	/**
	 * URL to a publisher location.
	 *
	 * @type String|null
	 */
	this.pubLocation = pubLocation;

	/**
	 * Callback to be invoked when a message is received through the subscription.
	 *
	 * @type Function
	 */
	this.onmessage = function() {};

	/**
	 * Callback to be invoked when the subscription connection is closed.
	 *
	 * @type Function
	 */
	this.onclose = function() {};
	
	/**
	 * SetupXHR initializes an XHR object and returns it.
	 *
	 * @param   {Function} callback
	 * @returns {XMLHttpRequest|ActiveXObject}
	 */
	function setupXHR(callback) {
	}
	
	/**
	 * ToQuery converts an object into query parameters.
	 *
	 * @param   {Object} object
	 * @returns {String}
	 */
	function toQuery(object) {
		var q = '', key;
		for (key in obj) {
			if (obj[key] !== null) {
				q += encodeURIComponent(key) + '=' + encodeURIComponent(obj[key]) + '&';
			}
		}
		return q;
	}
	
	/**
	 * XHR dispatchs an XHR request. When the request is done, the callback will be invoked
	 * with the received responseText, status and XHR object as the arguments.
	 *
	 * @param {String}   url
	 * @param {String}   method
	 * @param {Object}   options
	 * @param {Function} callback
	 */
	function request(url, method, options, callback) {
		var options = options || {},
			callback = callback || function() {},
			data = options.data || '',
			headers = options.headers || {},
			q, xhr, key;

		xhr = window.XMLHttpRequest ? new window.XMLHttpRequest() : new ActiveXObject('Microsoft.XMLHTTP');
		if (xhr == null) {
			throw 'Unable to initialize XHR object';
		}
		xhr.onreadystatechange = function() {
			if (xhr.readyState === 4) {
				callback(xhr);
				xhr = null;
			}
		};

		url = (url.indexOf('?') >= 0 ? url : url + '?') + '_=' + new Date().getTime();
		data = data.charAt ? data : toQuery(data);

		if (method === 'GET') {
			url = data == '' ? url : url + '&' + data;
			data = null;
		}
		
		xhr.open(method, url, true);
		for (key in headers) {
			xhr.setRequestHeader(key, headers[key]);
		}
		xhr.send(data);
	};
	
	/**
	 * Status dispatches a status request to the publisher location.
	 * When status is received, it invokes the callback with the received status
	 * as the sole argument. If request fails, a false will be passed to the callback.
	 *
	 * @param {String}   [contentType] The desired content type. Defaults to text/plain.
	 * @param {Function} callback
	 * @type  Function
	 */
	this.status = function(contentType, callback) {
		contentType = contentType || 'text/plain';
		if (!this.pubLocation) {
			throw 'Can\'t fetch status when publish location is not provided';
		}
		request(this.pubLocation, 'GET', {
			headers: { Accept: contentType }
		}, function(xhr) {
			if (xhr && xhr.status === 200) {
				callback((typeof JSON === 'object'
					&& xhr.getResponseHeader('Content-Type').indexOf('/json') > 0)
					? JSON.parse(xhr.responseText) : xhr.responseText);
			} else {
				callback(false);
			}
		});
	};
	
	/**
	 * Publish dispatches an publish request to the publisher location. Callback will
	 * be invoked with true or false as it's argument reflecting the result of the publishment.
	 *
	 * @param {String} [contentType]
	 * @param {data
	 * @param {Function} [callback]
	 * @type  Function
	 */
	this.publish = function(contentType, data, callback) {
		if (!this.pubLocation) {
			throw 'Can\'t publish when publish location is not provided';
		}
		request(this.pubLocation, 'POST', { headers: { 'Content-Type': contentType }, data: data }, function(xhr) {
			if (typeof callback !== 'function') {
				return;
			}
			if (xhr.status === 201 || xhr.status === 202) {
				callback(true);
			} else if (xhr.status === 0) {
				callback(false);
			}
		});
	};
	
	/**
	 * Subscribe initiates a long-polling connection to the subscriber location. It will
	 * keep on polling until an error is detected and then trigger the onerror callback
	 * of this object.
	 * On each received message, the onmessage callback of this object will be invoked.
	 *
	 * @param {String} [contentType]
	 * @param {data
	 * @param {Function} callback
	 * @type  Function
	 */
	this.subscribe = function() {
		if (!this.subLocation) {
			throw 'Can\'t subscribe when subscribe location is not provided';
		}

		function poll() {
			request(that.subLocation, 'GET', {
				headers: {
					'If-Modified-Since': since,
					'If-None-Match': etag
				}
			}, function(xhr) {
				// opera sends 0 when status is 304
				var status = xhr.status === 0 && window.opera ? 304 : xhr.status,
					error;
				switch (status) {
					case 304:
						break;
					case 200:
						etag = xhr.getResponseHeader('Etag');
						since = xhr.getResponseHeader('Last-Modified');
						etag = etag !== null ? parseInt(etag) : 0;
						since = since !== null ? since : 0;
						that.onmessage((typeof JSON === 'object'
							&& xhr.getResponseHeader('Content-Type').indexOf('/json') > 0)
							? JSON.parse(xhr.responseText) : xhr.responseText);
						break;
					case 409:
						error = 'Conflict';
						break;
					case 410:
						error = 'Channel was deleted';
						break;
					default:
						error = xhr.statusText || 'Unknown error';
						break;
				}
				if (error) {
					that.onclose(xhr.status + ': ' + error);
					return;
				}
				setTimeout(poll, 0);
			});
		}

		var that = this;
		poll();
	};
}
