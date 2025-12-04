// labels is a map of campaign statuses to
// CSS classes
var labels = {
    "In progress": "label-primary",
    "Queued": "label-info",
    "Completed": "label-success",
    "Emails Sent": "label-success",
    "Error": "label-danger"
}

var campaigns = []
var campaign = {}
var teams = []
var item_type = "campaigns"

const getDateISO = (selector, { format = "MMMM Do YYYY", endOfDay = false } = {}) => {
  const value = $(selector).val();
  if (!value) return null;

  const m = moment.utc(value, format);
  return (endOfDay ? m.endOf("day") : m.startOf("day")).toISOString();
};

const getTimeISO = (selector, { format = "h:mm a" } = {}) => {
  const value = $(selector).val();
  if (!value) return null;

  return moment.utc(value, format).format("2000-01-01[T]HH:mm:ss[Z]");
};

// Launch attempts to POST to /campaigns/
function launch() {
    Swal.fire({
        title: "Are you sure?",
        text: "This will schedule the campaign to be launched.",
        type: "question",
        animation: false,
        showCancelButton: true,
        confirmButtonText: "Launch",
        confirmButtonColor: "#428bca",
        reverseButtons: true,
        allowOutsideClick: false,
        showLoaderOnConfirm: true,
        preConfirm: function () {
            return new Promise(function (resolve, reject) {
                groups = []
                $("#users").select2("data").forEach(function (group) {
                    groups.push({
                        id: parseInt(group.id)
                    });
                })

                var selected = $("#scenario").select2("data")
                var scenarios = []

                for (var i = 0; i < selected.length; i++) {
                    scenarios.push({ id: parseInt($("#scenario").select2("data")[i].id) })
                }
                campaign = {
                    name: $("#name").val(),
                    scenarios: scenarios,
                    smtp: {
                        id: parseInt($("#profile").select2("data")[0].id)
                    },
                    launch_date: getDateISO("#launch_date"),
                    send_by_date: getDateISO("#send_by_date", { endOfDay: true }) || null,
                    groups: groups,
                    start_time : getTimeISO("#start_time") || null,
                    end_time : getTimeISO("#end_time") || null,
                    location : $("#timezone").val() || "UTC"
                }
                // Submit the campaign
                api.campaigns.post(campaign)
                    .success(function (data) {
                        resolve()
                        campaign = data
                    })
                    .error(function (data) {
                        $("#modal\\.flashes").empty().append("<div style=\"text-align:center\" class=\"alert alert-danger\">\
            <i class=\"fa fa-exclamation-circle\"></i> " + data.responseJSON.message + "</div>")
                        Swal.close()
                    })
            })
        }
    }).then(function (result) {
        if (result.value) {
            Swal.fire(
                'Campaign Scheduled!',
                'This campaign has been scheduled for launch!',
                'success'
            );
        }
        $('button:contains("OK")').on('click', function () {
            window.location = "/campaigns/" + campaign.id.toString()
        })
    })
}

// Attempts to send a test email by POSTing to /campaigns/
function sendTestEmail() {
    var selected_scenarios = $("#scenario").select2("data")

    for (var i = 0; i < selected_scenarios.length; i++) {
        api.scenarioId.get(parseInt($("#scenario").select2("data")[i].id))
        .success(function (scenario) {
            for (const template of scenario.templates) {
                var test_email_request = {
                    template: {
                        id: template.id
                    },
                    page: {
                        id: scenario.page.id
                    },
                    url: scenario.url,
                    first_name: $("input[name=to_first_name]").val(),
                    last_name: $("input[name=to_last_name]").val(),
                    email: $("input[name=to_email]").val(),
                    position: $("input[name=to_position]").val(),
                    smtp: {
                        id: parseInt($("#profile").select2("data")[0].id)
                    }
                }
                btnHtml = $("#sendTestModalSubmit").html()
                $("#sendTestModalSubmit").html('<i class="fa fa-spinner fa-spin"></i> Sending')
                // Send the test email
                api.send_test_email(test_email_request)
                    .success(function (data) {
                        $("#sendTestEmailModal\\.flashes").empty().append("<div style=\"text-align:center\" class=\"alert alert-success\">\
                        <i class=\"fa fa-check-circle\"></i> Email Sent!</div>")
                        $("#sendTestModalSubmit").html(btnHtml)
                    })
                    .error(function (data) {
                        $("#sendTestEmailModal\\.flashes").empty().append("<div style=\"text-align:center\" class=\"alert alert-danger\">\
                        <i class=\"fa fa-exclamation-circle\"></i> " + data.responseJSON.message + "</div>")
                        $("#sendTestModalSubmit").html(btnHtml)
                    })
            }
        })
        .error(function (data) {
            btnHtml = $("#sendTestModalSubmit").html()
            $("#sendTestModalSubmit").html('<i class="fa fa-spinner fa-spin"></i> Sending')

            $("#sendTestEmailModal\\.flashes").empty().append("<div style=\"text-align:center\" class=\"alert alert-danger\">\
            <i class=\"fa fa-exclamation-circle\"></i> " + "Please set the scenarios" + "</div>")
            $("#sendTestModalSubmit").html(btnHtml)
        })
    }
}

function dismiss() {
    $("#modal\\.flashes").empty();
    $("#name").val("");
    $("#profile").val("").change();
    $("#users").val("").change();
    $("#modal").modal('hide');
}

function deleteCampaign(idx) {
    Swal.fire({
        title: "Are you sure?",
        text: "This will delete the campaign. This can't be undone!",
        type: "warning",
        animation: false,
        showCancelButton: true,
        confirmButtonText: "Delete " + escapeHtml(campaigns[idx].name),
        confirmButtonColor: "#428bca",
        reverseButtons: true,
        allowOutsideClick: false,
        preConfirm: function () {
            return new Promise(function (resolve, reject) {
                api.campaignId.delete(campaigns[idx].id)
                    .success(function (msg) {
                        resolve()
                    })
                    .error(function (data) {
                        reject(data.responseJSON.message)
                    })
            }).catch(function (error) {
                Swal.showValidationMessage(
                    `Request failed: ${error}`
                );
                return false;
            })
        }
    }).then(function (result) {
        if (result.value) {
            Swal.fire(
                'Campaign Deleted!',
                'This campaign has been deleted!',
                'success'
            );
        }
        $('button:contains("OK")').on('click', function () {
            location.reload()
        })
    })
}

function setupOptions() {
    api.scenarios.get()
        .success(function (scenarios) {
            if (scenarios.length === 0) {
                modalError("No scenarios found!")
                return false
            } else {
                var scenario_s2 = $.map(scenarios, function (obj) {
                    obj.text = obj.name
                    return obj
                });
                var scenario_select = $("#scenario.form-control")
                scenario_select.select2({
                    placeholder: "Select the Scenarios",
                    data: scenario_s2,
                }).select2("val", scenario_s2[0]);
                if (scenarios.length === 1) {
                    scenario_select.val(scenario_select[0].id)
                    scenario_select.trigger('change.select2')
                }
            }
        });
    api.groups.summary()
        .success(function (summaries) {
            groups = summaries.groups
            if (groups.length == 0) {
                modalError("No groups found!")
                return false;
            } else {
                var group_s2 = $.map(groups, function (obj) {
                    obj.text = obj.name
                    obj.title = obj.num_targets + " targets"
                    return obj
                });
                console.log(group_s2)
                $("#users.form-control").select2({
                    placeholder: "Select Groups",
                    data: group_s2,
                });
            }
        });
    api.SMTP.get()
        .success(function (profiles) {
            if (profiles.length == 0) {
                modalError("No profiles found!")
                return false
            } else {
                var profile_s2 = $.map(profiles, function (obj) {
                    obj.text = obj.name
                    return obj
                });
                var profile_select = $("#profile.form-control")
                profile_select.select2({
                    placeholder: "Select a Sending Profile",
                    data: profile_s2,
                }).select2("val", profile_s2[0]);
                if (profiles.length === 1) {
                    profile_select.val(profile_s2[0].id)
                    profile_select.trigger('change.select2')
                }
            }
        });
    const select = document.getElementById("timezone");
    if (!select) return;

    // Try to detect user's local timezone
    const localTz = Intl.DateTimeFormat().resolvedOptions().timeZone;

    const timezones = Intl.supportedValuesOf 
        ? Intl.supportedValuesOf("timeZone")
        : [
            "UTC","Europe/London","Europe/Berlin","Europe/Paris",
            "Asia/Shanghai","Asia/Tokyo","Asia/Singapore",
            "America/New_York","America/Chicago","America/Los_Angeles",
            "Australia/Sydney"
            ];

    timezones.forEach(tz => {
        const opt = document.createElement("option");
        opt.value = tz;
        opt.textContent = tz;
        if (tz === localTz) opt.selected = true;
        select.appendChild(opt);
    });
}

function edit(campaign) {
    setupOptions();
}

function copy(idx) {
    setupOptions();
    // Set our initial values
    api.campaignId.get(campaigns[idx].id)
        .success(function (campaign) {
            $("#name").val("Copy of " + campaign.name)
            var campaign_scenarios = []
            console.log(campaign.scenarios)
            campaign.scenarios.forEach((item) => {
                if (item.hasOwnProperty('id')) {
                    campaign_scenarios.push(item.id.toString())
                }
            })
            if (campaign_scenarios.length === 0){
                $("#scenario").val("").change();
                $("#scenario").select2({
                    placeholder: "Add Scenarios"
                });
            } else {
                $("#scenario").val(campaign_scenarios);
                $("#scenario").trigger("change.select2")
            }
            if (!campaign.smtp.id) {
                $("#profile").val("").change();
                $("#profile").select2({
                    placeholder: campaign.smtp.name
                });
            } else {
                $("#profile").val(campaign.smtp.id.toString());
                $("#profile").trigger("change.select2")
            }
        })
        .error(function (data) {
            $("#modal\\.flashes").empty().append("<div style=\"text-align:center\" class=\"alert alert-danger\">\
            <i class=\"fa fa-exclamation-circle\"></i> " + data.responseJSON.message + "</div>")
        })
}

$(document).ready(function () {
    $("#launch_date").datetimepicker({
        "widgetPositioning": {
            "vertical": "bottom"
        },
        "showTodayButton": true,
        "defaultDate": moment(),
        "format": "MMMM Do YYYY"
    })
    $("#send_by_date").datetimepicker({
        "widgetPositioning": {
            "vertical": "bottom"
        },
        "showTodayButton": true,
        "useCurrent": false,
        "format": "MMMM Do YYYY"
    })
    $("#start_time").datetimepicker({
        "widgetPositioning": {
            "vertical": "bottom"
        },
        "showTodayButton": false,
        "defaultDate": false,
        "format": "h:mm a"
    })
    $("#end_time").datetimepicker({
        "widgetPositioning": {
            "vertical": "bottom"
        },
        "showTodayButton": false,
        "defaultDate": false,
        "format": "h:mm a"
    })
    if (!document.getElementById('special_time_checkbox').checked) {
        $(".toggle-label-time").hide()
        $("#start_time").hide()
        $("#end_time").hide()
    }
    if (document.getElementById('special_sending_checkbox').checked) {
        $(".toggle-label").hide()
        $("#send_by_date").hide()
    }
    $("#special_time_checkbox").change(function () {
    	if (document.getElementById('special_time_checkbox').checked) {
    		$(".toggle-label-time").show()
    		$("#start_time").show()
            $("#end_time").show()
        } else {
        	$(".toggle-label-time").hide()
        	$("#start_time").hide()
            $("#end_time").hide()
            $("#start_time").data("DateTimePicker").date(null)
            $("#end_time").data("DateTimePicker").date(null) 
        }
    })
    $("#special_sending_checkbox").change(function () {
    	if (document.getElementById('special_sending_checkbox').checked) {
    		$(".toggle-label").hide()
    		$("#send_by_date").hide()
    		$("#launch_date").datetimepicker().data('DateTimePicker').format('MMMM Do YYYY, h:mm a');
    		$("#send_by_date").data("DateTimePicker").date(null)

        } else {
        	$(".toggle-label").show()
        	$("#send_by_date").show()
		$("#launch_date").datetimepicker().data('DateTimePicker').format('MMMM Do YYYY');
    		$("#send_by_date").datetimepicker().data('DateTimePicker').format('MMMM Do YYYY');   
        }
    })
    // Setup multiple modals
    // Code based on http://miles-by-motorcycle.com/static/bootstrap-modal/index.html
    $('.modal').on('hidden.bs.modal', function (event) {
        $(this).removeClass('fv-modal-stack');
        $('body').data('fv_open_modals', $('body').data('fv_open_modals') - 1);
    });
    $('.modal').on('shown.bs.modal', function (event) {
        // Keep track of the number of open modals
        if (typeof ($('body').data('fv_open_modals')) == 'undefined') {
            $('body').data('fv_open_modals', 0);
        }
        // if the z-index of this modal has been set, ignore.
        if ($(this).hasClass('fv-modal-stack')) {
            return;
        }
        $(this).addClass('fv-modal-stack');
        // Increment the number of open modals
        $('body').data('fv_open_modals', $('body').data('fv_open_modals') + 1);
        // Setup the appropriate z-index
        $(this).css('z-index', 1040 + (10 * $('body').data('fv_open_modals')));
        $('.modal-backdrop').not('.fv-modal-stack').css('z-index', 1039 + (10 * $('body').data('fv_open_modals')));
        $('.modal-backdrop').not('fv-modal-stack').addClass('fv-modal-stack');
    });
    // Scrollbar fix - https://stackoverflow.com/questions/19305821/multiple-modals-overlay
    $(document).on('hidden.bs.modal', '.modal', function () {
        $('.modal:visible').length && $(document.body).addClass('modal-open');
    });
    $('#modal').on('hidden.bs.modal', function (event) {
        dismiss()
    });
    api.campaigns.summary()
        .success(function (data) {
            api.user.current().success(function (u) {
                teams = u.teams
                items = data.campaigns
                campaigns = data.campaigns
                $("#loading").hide()
                if (campaigns.length > 0) {
                    $("#campaignTable").show()
                    $("#campaignTableArchive").show()

                    activeCampaignsTable = $("#campaignTable").DataTable({
                        columnDefs: [{
                            orderable: false,
                            targets: "no-sort"
                        }],
                        order: [
                            [1, "desc"]
                        ]
                    });
                    archivedCampaignsTable = $("#campaignTableArchive").DataTable({
                        columnDefs: [{
                            orderable: false,
                            targets: "no-sort"
                        }],
                        order: [
                            [1, "desc"]
                        ]
                    });
                    rows = {
                        'active': [],
                        'archived': []
                    }
                    campaigns = campaigns.filter((item, index, self) =>
                        index === self.findIndex((t) => t.id === item.id)
                    );

                    $.each(campaigns, function (i, campaign) {
                        var permissions = {
                            canDelete : false,
                            canEdit : false,
                        };
                        permissions = CheckTeam(campaign.teams, u)

                        var isOwner = false;
                        if (u.id == campaign.user_id) {
                            var isOwner = true;
                        }
                        label = labels[campaign.status] || "label-default";

                        //section for tooltips on the status of a campaign to show some quick stats
                        var launchDate;
                        if (moment(campaign.launch_date).isAfter(moment())) {
                            launchDate = "Scheduled to start: " + moment(campaign.launch_date).format('MMMM Do YYYY, h:mm:ss a')
                            var quickStats = launchDate + "<br><br>" + "Number of recipients: " + campaign.stats.total
                        } else {
                            launchDate = "Launch Date: " + moment(campaign.launch_date).format('MMMM Do YYYY, h:mm:ss a')
                            var quickStats = launchDate + "<br><br>" + "Number of recipients: " + campaign.stats.total + "<br><br>" + "Emails opened: " + campaign.stats.opened + "<br><br>" + "Emails clicked: " + campaign.stats.clicked + "<br><br>" + "Submitted Credentials: " + campaign.stats.submitted_data + "<br><br>" + "Errors : " + campaign.stats.error + "<br><br>" + "Reported : " + campaign.stats.email_reported
                        }

                        var row = [
                            escapeHtml(campaign.name),
                            moment(campaign.created_date).format('MMMM Do YYYY, h:mm:ss a'),
                            "<span class=\"label " + label + "\" data-toggle=\"tooltip\" data-placement=\"right\" data-html=\"true\" title=\"" + quickStats + "\">" + campaign.status + "</span>",
                            "<div class='pull-right'><a class='btn btn-primary' href='/campaigns/" + campaign.id + "' data-toggle='tooltip' data-placement='left' title='View Results'>\
                    <i class='fa fa-bar-chart'></i>\
                    </a>\
            <span data-toggle='modal' data-backdrop='static' data-target='#modal'><button class='btn btn-primary' data-toggle='tooltip' data-placement='left' title='Copy Campaign' onclick='copy(" + i + ")'>\
                    <i class='fa fa-copy'></i>\
                    </button></span>\
                    <button class='btn " + (isOwner || permissions.canDelete ? "btn-danger" : "btn-secondary disabled") + "' data-toggle='tooltip' data-placement='left' title='" + (isOwner || permissions.canDelete ? "Delete Campaign" : "You dont have permission to delete this campaign") + "' " + (isOwner || permissions.canDelete ? "onclick='deleteCampaign(" + i + ")'" : "disabled") + ">\
                    <i class='fa fa-trash-o'></i>\
                    </button>\
                    <button id='teams_button' type='button' class='btn "+ (isOwner ? "btn-orange" : "btn-secondary disabled") + "' data-toggle='modal' data-backdrop='static' data-target='#team_modal'" + (isOwner ? "onclick='team("+ i + ")'" : "disabled") + "'>\
                    <i class='fa fa-users'></i> Update teams\
                    </button></div>"
                        ]
                        if (campaign.status == 'Completed') {
                            rows['archived'].push(row)
                        } else {
                            rows['active'].push(row)
                        }
                    })
                    activeCampaignsTable.rows.add(rows['active']).draw()
                    archivedCampaignsTable.rows.add(rows['archived']).draw()
                    $('[data-toggle="tooltip"]').tooltip()
                } else {
                    $("#emptyMessage").show()
                }
            })
        })
        .error(function () {
            $("#loading").hide()
            errorFlash("Error fetching campaigns")
        })
    // Select2 Defaults
    $.fn.select2.defaults.set("width", "100%");
    $.fn.select2.defaults.set("dropdownParent", $("#modal_body"));
    $.fn.select2.defaults.set("theme", "bootstrap");
    $.fn.select2.defaults.set("sorter", function (data) {
        return data.sort(function (a, b) {
            if (a.text.toLowerCase() > b.text.toLowerCase()) {
                return 1;
            }
            if (a.text.toLowerCase() < b.text.toLowerCase()) {
                return -1;
            }
            return 0;
        });
    })
})

function team(i) {
    updateItemTeamsAssignment(campaigns[i], item_type, teams)
}