<script lang="ts">
  import "bulma/css/bulma.css";
  type loginType = {
    Id?: string;
    Password?: string;
  };
  let message: string = "";
  let dat: loginType = { Id: "", Password: "" };

  const clickSubmit = () => {
    document.querySelector(".notification")?.classList.add("is-hidden");
    fetch("/auth/api/login", {
      method: "post",
      cache: "no-cache",
      body: JSON.stringify(dat),
    })
      .then((r) => {
        if (r.status == 200) {
          window.location.href = "/";
          return;
        }

        document.querySelector(".notification")?.classList.remove("is-hidden");
        if (r.status === 401) {
          message = "ID / PASSWORD do not match.";
        } else if (r.status === 400) {
          message = "ID / PASSWORD input required.";
        } else {
          message = "login failed.";
        }
      })
      .catch((r) => {
        message = "login failed.";
      });
  };
</script>

<main>

  <div class="container">
    <div class="card px-5">
      <header class="card-header px-10">
        <p class="card-header-title">login</p>
      </header>
      <div class="card-content">
        <div class="notification is-danger is-light is-hidden">
          {message}
        </div>
        <div class="content p-10">
          <div class="field is-horizontal">
            <div class="field-label is-normal">
              <label class="label" for="id">ID</label>
            </div>
            <div class="field-body">
              <div class="field">
                <p class="control">
                  <input
                    type="text"
                    class="input is-primary"
                    name="id"
                    bind:value={dat.Id}
                  />
                </p>
              </div>
            </div>
          </div>
          <div class="field is-horizontal">
            <div class="field-label is-normal">
              <label class="label" for="password">PASSWORD</label>
            </div>
            <div class="field-body">
              <div class="field">
                <p class="control">
                  <input
                    type="password"
                    class="input is-primary is-fullwidth p-2"
                    name="password"
                    bind:value={dat.Password}
                  />
                </p>
              </div>
            </div>
          </div>
        </div>
      </div>
      <div class="card-footer column is-8 is-offset-2">
        <div class="field">
          <div class="control">
            <button
              type="button"
              class="button is-primary"
              on:click={clickSubmit}>submit</button
            >
          </div>
        </div>
      </div>
    </div>
  </div>
</main>
