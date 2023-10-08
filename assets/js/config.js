tstor.config = {
  _editor: null,
  _infoDiv: document.getElementById("tstor-reload-info-text"),
  _loadingInfoDom: document.getElementById("tstor-reload-info-loading"),
  _valid: function () {
    if (this._editor == null) {
      return false;
    }

    let getYamlCodeValidationErrors = (code) => {
      var error = "";
      try {
        jsyaml.safeLoad(code);
      } catch (e) {
        error = e;
      }
      return error;
    };

    let code = this._editor.getValue();
    let error = getYamlCodeValidationErrors(code);
    if (error) {
      this._editor.getSession().setAnnotations([
        {
          row: error.mark.line,
          column: error.mark.column,
          text: error.reason,
          type: "error",
        },
      ]);

      return false;
    } else {
      this._editor.getSession().setAnnotations([]);

      return true;
    }
  },

  save: function () {
    fetch("/api/config", {
      method: "POST",
      body: this._editor.getValue(),
    })
      .then(function (response) {
        if (response.ok) {
          tstor.message.info("Configuration saved");
        } else {
          tstor.message.error(
            "Error saving configuration file. Response: " + response.status
          );
        }
      })
      .catch(function (error) {
        tstor.message.error(
          "Error saving configuration file: " + error.message
        );
      });
  },

  reload: function () {
    this.cleanInfo();
    fetch("/api/reload", {
      method: "POST",
    })
      .then(function (response) {
        if (response.ok) {
          return response.json();
        } else {
          tstor.config.showInfo(
            "Error reloading server. Response: " + response.status,
            "ko"
          );
        }
      })
      .then(function (json) {
        tstor.config.showInfo(json.message, "ok");
      })
      .catch(function (error) {
        tstor.message.error("Error reloading server: " + error.message);
      });
  },

  cleanInfo: function () {
    this._loadingInfoDom.style.display = "block";
    this._infoDiv.innerText = "";
  },

  showInfo: function (message, flag) {
    const li = document.createElement("li");
    li.innerText = message;
    li.className = "list-group-item";
    if (flag == "ok") {
      li.className += " list-group-item-success";
    } else if (flag == "ko") {
      li.className += " list-group-item-danger";
    }

    if (flag) {
      this._loadingInfoDom.style.display = "none";
    }

    this._infoDiv.appendChild(li);
  },

  loadView: function () {
    this._editor = ace.edit("editor");
    this._editor.getSession().setMode("ace/mode/yaml");
    this._editor.setShowPrintMargin(false);
    this._editor.setOptions({
      enableBasicAutocompletion: true,
      enableSnippets: true,
      enableLiveAutocompletion: false,

      autoScrollEditorIntoView: true,
      fontSize: "16px",
      maxLines: 100,
      wrap: true,
    });

    this._editor.commands.addCommand({
      name: "save",
      bindKey: { win: "Ctrl-S", mac: "Command-S" },
      exec: function (editor) {
        if (tstor.config._valid()) {
          tstor.config.save();
        } else {
          tstor.message.error("Check file format errors before saving");
        }
      },
      readOnly: false,
    });

    this._editor.on("change", () => {
      tstor.config._valid();
    });

    fetch("/api/config")
      .then(function (response) {
        if (response.ok) {
          return response.text();
        } else {
          tstor.message.error(
            "Error getting data from server. Response: " + response.status
          );
        }
      })
      .then(function (yaml) {
        tstor.config._editor.setValue(yaml);
      })
      .catch(function (error) {
        tstor.message.error("Error getting yaml from server: " + error.message);
      });

    var stream = new EventSource("/api/events");
    stream.addEventListener("event", function (e) {
      tstor.config.showInfo(e.data);
    });
  },
};
