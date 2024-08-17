function refreshServiceStatus() {
    window.go.main.App.GetServiceStatus().then(serviceStatus => {
        $("div#setup").hide();
        $("div#status").hide();

        if (serviceStatus == "NotInstalled") {
            $("div#setup").show();
        } else {
            if (serviceStatus == "Stopped") {
                var statusBar = `<div class="btn-toolbar" role="toolbar">
                        <div class="btn-group btn-group-lg me-2" role="group">
                            <button id="start" style="width:225px;height:225px;font-size:2em;" type="button" class="btn btn-outline-success"><i class="fa-sharp-duotone fa-solid fa-circle-play"></i> Start</button>
                            <button id="pause" style="width:225px;height:225px;font-size:2em;" type="button" class="btn btn-outline-warning" disabled><i class="fa-sharp-duotone fa-solid fa-circle-pause"></i> Pause</button>
                            <button id="stop" style="width:225px;height:225px;font-size:2em;" type="button" class="btn btn-danger" disabled><i class="fa-sharp-duotone fa-solid fa-circle-stop"></i> Stopped</button>
                            <button id="uninstall" style="width:225px;height:225px;font-size:2em;" type="button" class="btn btn-outline-danger"><i class="fa-sharp-duotone fa-solid fa-circle-x"></i> Uninstall</button>
                        </div>
                    </div>`;

                $("span#currentServiceStatus").html(serviceStatus);
                $("div#statusControl").html(statusBar);
                console.log($("#statusControl").find("#stop").css('background-color'));
                $("span#currentServiceStatus").css('background-color', $("#statusControl").find("#stop").css('background-color'));
                console.log($("#currentServiceStatus").css('background-color'));
                $("span#currentServiceStatus").html(serviceStatus);
                console.log($("#statusControl").find("#stop").css('background-color'))
            } else if (serviceStatus == "Running") {
                var statusBar = `<div class="btn-toolbar" role="toolbar">
                        <div class="btn-group btn-group-lg me-2" role="group">
                            <button id="start" style="width:225px;;height:225px;font-size:2em;" type="button" class="btn btn-success" disabled><i class="fa-sharp-duotone fa-solid fa-circle-play"></i> Running</button>
                            <button id="pause" style="width:225px;height:225px;font-size:2em;" type="button" class="btn btn-outline-warning"><i class="fa-sharp-duotone fa-solid fa-circle-pause"></i> Pause</button>
                            <button id="stop" style="width:225px;height:225px;font-size:2em;" type="button" class="btn btn-outline-danger"><i class="fa-sharp-duotone fa-solid fa-circle-stop"></i> Stop</button>
                            <button id="uninstall" style="width:225px;height:225px;font-size:2em;" type="button" class="btn btn-outline-danger disabled"><i class="fa-sharp-duotone fa-solid fa-circle-x"></i> Uninstall</button>
                        </div>
                    </div>`;
                $("span#currentServiceStatus").html(serviceStatus);
                $("div#statusControl").html(statusBar);
                console.log($("#statusControl").find("#start").css('background-color'));
                $("span#currentServiceStatus").css('background-color', $("#statusControl").find("#start").css('background-color'));
                console.log($("#currentServiceStatus").css('background-color'));
                $("span#currentServiceStatus").html(serviceStatus);
                console.log($("#statusControl").find("#start").css('background-color'))
            } else if (serviceStatus == "Paused") {
                var statusBar = `<div class="btn-toolbar" role="toolbar">
                        <div class="btn-group btn-group-lg me-2" role="group">
                            <button id="continue" style="width:225px;height:225px;font-size:2em;" type="button" class="btn btn-outline-success"><i class="fa-sharp-duotone fa-solid fa-circle-play"></i> Continue</button>
                            <button id="pause" style="width:225px;height:225px;font-size:2em;" type="button" class="btn btn-warning" disabled><i class="fa-sharp-duotone fa-solid fa-circle-pause"></i> Paused</button>
                            <button id="stop" style="width:225px;height:225px;font-size:2em;" type="button" class="btn btn-outline-danger"><i class="fa-sharp-duotone fa-solid fa-circle-stop"></i> Stop</button>
                            <button id="uninstall" style="width:225px;height:225px;font-size:2em;" type="button" class="btn btn-outline-danger"><i class="fa-sharp-duotone fa-solid fa-circle-x"></i> Uninstall</button>
                        </div>
                    </div>`;
                $("span#currentServiceStatus").html(serviceStatus);
                $("div#statusControl").html(statusBar);
                console.log($("#statusControl").find("#pause").css('background-color'));
                $("span#currentServiceStatus").css('background-color', $("#statusControl").find("#pause").css('background-color'));
                console.log($("#currentServiceStatus").css('background-color'));
                $("span#currentServiceStatus").html(serviceStatus);

            }
            $("div#status").show();
        }
    }).catch(error => {
        console.error(error);
    });
}

function updateStatusBar(serviceStatus, startClass, stopClass, statusText, isPaused = false) {
    const statusBar = `<div class="btn-toolbar" role="toolbar">
        <div class="btn-group me-2" role="group">
            <button id="start" style="width:150px;" type="button" class="btn ${startClass}" ${statusText !== "Stopped" ? "disabled" : ""}>
                <i class="fa-sharp-duotone fa-solid fa-circle-play"></i> ${statusText}
            </button>
            <button id="${isPaused ? 'continue' : 'pause'}" style="width:150px;" type="button" class="btn btn-outline-warning" ${!isPaused ? "" : "disabled"}>
                <i class="fa-sharp-duotone fa-solid fa-circle-${isPaused ? 'play' : 'pause'}"></i> ${isPaused ? 'Continue' : 'Pause'}
            </button>
            <button id="stop" style="width:150px;" type="button" class="btn ${stopClass}">
                <i class="fa-sharp-duotone fa-solid fa-circle-stop"></i> Stop
            </button>
            <button id="uninstall" style="width:150px;" type="button" class="btn btn-outline-danger">
                <i class="fa-sharp-duotone fa-solid fa-circle-x"></i> Uninstall
            </button>
        </div>
    </div>`;

    $("span#currentServiceStatus").html(serviceStatus);
    $("div#statusControl").html(statusBar);
    const bgColor = $("#statusControl").find(`#${statusText.toLowerCase() === 'stopped' ? 'stop' : 'start'}`).css('background-color');
    $("span#currentServiceStatus").css('background-color', bgColor);
}

$(document).ready(function () {
    refreshServiceStatus();

    $("#statusControl").on("click", "#start, #continue, #pause, #stop, #uninstall", function () {
        const command = $(this).attr('id');
        window.go.main.App.ServiceControl(command).then(message => {
            console.log(message);
            if (command === "uninstall") {
                $("div#status").hide();
                $("div#setup").show();
            }
            refreshServiceStatus();
        }).catch(error => {
            console.error(`Error setting service status: ${error}`);
        });
    });

    $('#browseBackupFolder').click(function () {
        window.go.main.App.BrowseFolder().then(folder => {
            $('#backupFolder').val(folder);
        }).catch(error => {
            console.error("Error browsing folder:", error);
        });
    });

    $('#browseLogFolder').click(function () {
        window.go.main.App.BrowseFolder().then(folder => {
            $('#logFolder').val(folder);
        }).catch(error => {
            console.error("Error browsing folder:", error);
        });
    });

    $('#backupForm').submit(function (event) {
        event.preventDefault();
        const formData = {
            backupFolder: $('#backupFolder').val(),
            logFolder: $('#logFolder').val(),
            minioEndpoint: $('#minioEndpoint').val(),
            minioKey: $('#minioKey').val(),
            minioSecret: $('#minioSecret').val(),
            minioBucketName: $('#minioBucketName').val(),
            backupFrequencySeconds: $('#backupFrequencySeconds').val()
        };
        window.go.main.App.SubmitForm(formData).then(response => {
            console.log("Form submitted successfully:", response);
            refreshServiceStatus();
        }).catch(error => {
            console.error("Error submitting form:", error);
            alert("Error submitting form!");
        });
    });
});
