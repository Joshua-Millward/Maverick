function ListenerUI(mode_create)
{
    // === MAIN SETTINGS ===
    let labelHost = form.create_label("Host & port (Bind):");
    let comboHostBind = form.create_combo();
    comboHostBind.setEnabled(mode_create);
    comboHostBind.clear();
    let addrs = ax.interfaces();
    for (let item of addrs) { comboHostBind.addItem(item); }

    let spinPortBind = form.create_spin();
    spinPortBind.setRange(1, 65535);
    spinPortBind.setValue(443);
    spinPortBind.setEnabled(mode_create);

    let labelUri = form.create_label("URI:");
    let textUri = form.create_textline("/");
    textUri.setPlaceholder("/callback");

    // === SSL SETTINGS ===
    let certSelector = form.create_selector_file();
    certSelector.setPlaceholder("SSL certificate");

    let keySelector = form.create_selector_file();
    keySelector.setPlaceholder("SSL key");

    let ssl_layout = form.create_gridlayout();
    ssl_layout.addWidget(certSelector,    0, 0, 1, 3);
    ssl_layout.addWidget(keySelector,     1, 0, 1, 3);

    let ssl_panel = form.create_panel();
    ssl_panel.setLayout(ssl_layout);

    let ssl_group = form.create_groupbox("Use SSL (HTTPS)", true);
    ssl_group.setPanel(ssl_panel);
    ssl_group.setChecked(false);

    // === MAIN LAYOUT ===
    let layoutMain = form.create_gridlayout();
    layoutMain.addWidget(labelHost,     0, 0, 1, 1);
    layoutMain.addWidget(comboHostBind, 0, 1, 1, 1);
    layoutMain.addWidget(spinPortBind,  0, 2, 1, 1);
    layoutMain.addWidget(labelUri,      1, 0, 1, 1);
    layoutMain.addWidget(textUri,       1, 1, 1, 2);
    layoutMain.addWidget(ssl_group,     2, 0, 1, 3);

    let panelMain = form.create_panel();
    panelMain.setLayout(layoutMain);

    // === CONTAINER ===
    let container = form.create_container();
    container.put("host_bind", comboHostBind);
    container.put("port_bind", spinPortBind);
    container.put("uri", textUri);
    container.put("ssl", ssl_group);
    container.put("ssl_cert", certSelector);
    container.put("ssl_key", keySelector);

    let layout = form.create_hlayout();
    layout.addWidget(panelMain);

    let panel = form.create_panel();
    panel.setLayout(layout);

    return {
        ui_panel: panel,
        ui_container: container
    }
}
