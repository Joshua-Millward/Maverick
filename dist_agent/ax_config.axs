/// Maverick agent

let exit_thread_action  = menu.create_action("Terminate thread",  function(value) { value.forEach(v => ax.execute_command(v, "exit thread")) });
let exit_process_action = menu.create_action("Terminate process", function(value) { value.forEach(v => ax.execute_command(v, "exit process")) });
let exit_menu = menu.create_menu("Exit");
exit_menu.addItem(exit_thread_action)
exit_menu.addItem(exit_process_action)
menu.add_session_agent(exit_menu, ["maverick"])


function RegisterCommands(listenerType)
{
    let cmd_whoami = ax.create_command("whoami", "Display the current username", "whoami", "Task: retrieve current user identity");

    let cmd_sleep = ax.create_command("sleep", "Set the callback sleep interval", "sleep 5s", "Task: update callback interval");
    cmd_sleep.addArgString("val", true, "Sleep time (e.g., 3s, 10s, 1m)");

    let cmd_exit_thread = ax.create_command("thread", "Exit agent thread (process stays alive)", "exit thread", "Task: terminate agent thread");
    let cmd_exit_process = ax.create_command("process", "Exit agent and kill process", "exit process", "Task: terminate agent process");
    let cmd_exit = ax.create_command("exit", "Terminate the agent");
    cmd_exit.addSubCommands([cmd_exit_thread, cmd_exit_process]);

    if (listenerType == "MaverickHTTP") {
        let commands = ax.create_commands_group("maverick", [cmd_whoami, cmd_sleep, cmd_exit]);
        return { commands_windows: commands }
    }

    return ax.create_commands_group("none", []);
}

function GenerateUI(listenerType)
{
    // === Output Settings ===

    let labelFormat = form.create_label("Compilation Format:");
    let comboFormat = form.create_combo();
    comboFormat.addItem("Exe");
    comboFormat.addItem("Dll");
    comboFormat.addItem("Svc");
    comboFormat.addItem("Bin");
    comboFormat.setCurrentIndex(0);

    let debugCheck = form.create_check("Console subsystem (debug)");
    debugCheck.setChecked(false);

    let layout_output = form.create_gridlayout();
    layout_output.addWidget(labelFormat,  0, 0, 1, 1);
    layout_output.addWidget(comboFormat,  0, 1, 1, 1);
    layout_output.addWidget(debugCheck,   1, 0, 1, 2);

    let panel_output = form.create_panel();
    panel_output.setLayout(layout_output);
    let output_group = form.create_groupbox("Output", false);
    output_group.setPanel(panel_output);

    // === Callback Settings ===

    let labelSleep = form.create_label("Sleep:");
    let textSleep = form.create_textline("3s");
    textSleep.setPlaceholder("e.g., 5s, 10s");

    let labelJitter = form.create_label("Jitter (%):");
    let spinJitter = form.create_spin();
    spinJitter.setRange(0, 100);
    spinJitter.setValue(0);

    let layout_callback = form.create_gridlayout();
    layout_callback.addWidget(labelSleep,     0, 0, 1, 1);
    layout_callback.addWidget(textSleep,      0, 1, 1, 1);
    layout_callback.addWidget(labelJitter,    1, 0, 1, 1);
    layout_callback.addWidget(spinJitter,     1, 1, 1, 1);

    let panel_callback = form.create_panel();
    panel_callback.setLayout(layout_callback);
    let callback_group = form.create_groupbox("Callback", false);
    callback_group.setPanel(panel_callback);

    // === Main Layout ===

    let layout_scroll = form.create_gridlayout();
    layout_scroll.addWidget(output_group,    0, 0, 1, 1);
    layout_scroll.addWidget(callback_group,  1, 0, 1, 1);

    let panel_scroll = form.create_panel();
    panel_scroll.setLayout(layout_scroll);

    const scroll = form.create_scrollarea();
    scroll.setPanel(panel_scroll);

    let layout = form.create_gridlayout();
    layout.addWidget(scroll, 0, 0, 1, 1);

    let container = form.create_container()
    container.put("format", comboFormat)
    container.put("sleep", textSleep)
    container.put("jitter", spinJitter)
    container.put("debug", debugCheck)

    let panel = form.create_panel()
    panel.setLayout(layout)

    return {
        ui_panel: panel,
        ui_container: container,
        ui_height: 600,
        ui_width: 800
    }
}
