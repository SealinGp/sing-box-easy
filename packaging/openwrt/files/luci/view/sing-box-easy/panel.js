'use strict';
'require view';
'require uci';

/*
 * LuCI entry for the sing-box-easy panel.
 *
 * The panel is a standalone HTTP server on its own port, not a LuCI app, so
 * this view is a thin shell: it embeds the panel in an iframe and always
 * offers a plain link out, because embedding can fail for reasons this page
 * cannot detect from the outside (see below).
 *
 * The host is taken from the address the operator is already using rather than
 * being hardcoded, so the link works over LAN IP, hostname or VPN alike. The
 * port comes from UCI (seeded from app.yml at install time) so that changing
 * server.port does not require editing this file.
 */

var DEFAULT_PORT = '8080';

function panelURL() {
	var port = uci.get('sing-box-easy', 'main', 'port') || DEFAULT_PORT;
	return window.location.protocol + '//' + window.location.hostname + ':' + port + '/';
}

return view.extend({
	load: function () {
		// A missing config is normal on a fresh install; fall back to the default.
		return uci.load('sing-box-easy').catch(function () { return null; });
	},

	render: function () {
		var url = panelURL();

		var openLink = E('a', {
			'href': url,
			'target': '_blank',
			'rel': 'noopener',
			'class': 'cbi-button cbi-button-apply'
		}, _('Open sing-box-easy'));

		/*
		 * The iframe is best-effort. A browser will refuse it when LuCI is
		 * served over https and the panel over plain http (mixed content), and
		 * some setups place the panel behind a proxy that sends
		 * X-Frame-Options. Neither is detectable from here, so the link above
		 * is always present as the reliable path.
		 */
		var frame = E('iframe', {
			'src': url,
			'style': 'width:100%; height:80vh; border:1px solid #ccc; border-radius:4px; background:#fff;'
		});

		return E('div', { 'class': 'cbi-map' }, [
			E('h2', {}, 'sing-box-easy'),
			E('div', { 'class': 'cbi-map-descr' }, [
				_('Manage sing-box configuration, subscriptions and the service lifecycle.'),
				' ',
				E('span', { 'style': 'white-space:nowrap' }, [
					_('Panel address:'), ' ',
					E('code', {}, url)
				])
			]),
			E('p', {}, openLink),
			frame
		]);
	},

	// Read-only shell: LuCI's save/apply footer would be meaningless here.
	handleSave: null,
	handleSaveApply: null,
	handleReset: null
});
