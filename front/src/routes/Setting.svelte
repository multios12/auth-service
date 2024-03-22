<script lang="ts">
  import "bulma/css/bulma.css";
  let message: string;
  let dat = { OldPassword: "", NewPassword: "", ConfirmPassword: "" };

  const clickSubmit = () => {
    document.querySelector(".notification")?.classList.add("is-hidden");

    if (dat.NewPassword != dat.ConfirmPassword) {
      message = "Password confirmation doesn't match the password";
      return;
    }

    fetch("/auth/api/info", {
      method: "post",
      cache: "no-cache",
      body: JSON.stringify(dat),
    })
      .then((r) => {
        if (r.status != 200) {
          message = "password change failed.";          
        } else {

        }
      })
      .catch((r) => {
        message = "password change failed.";
      });
  };
</script>

<main>
  <div class="container">
    <div class="card">
      <header class="card-header px-10">
        <p class="card-header-title">change password</p>
      </header>
      <div class="card-content">
        <div class="notification is-danger is-light is-hidden">
          {message}
        </div>
        <div class="content">
          <div class="field is-horizontal">
            <div class="field-label is-normal">
              <label class="label" for="oldPassword">old password</label>
            </div>
            <div class="field">
              <div class="control">
                <input
                  type="password"
                  name="oldPassword"
                  bind:value={dat.OldPassword}
                />
              </div>
            </div>
          </div>
          <div class="field is-horizontal">
            <div class="field-label is-normal">
              <label class="label" for="newPassword">new password</label>
            </div>
            <div class="field">
              <div class="control">
                <input
                  type="password"
                  name="newPassword"
                  bind:value={dat.NewPassword}
                />
              </div>
            </div>
          </div>
          <div class="field is-horizontal">
            <div class="field-label is-normal">
              <label class="label" for="rePassword">reInput password</label>
            </div>
            <div class="field">
              <div class="control">
                <input
                  type="password"
                  name="rePassword"
                  bind:value={dat.ConfirmPassword}
                />
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
              on:click={clickSubmit}
              class="button is-primary">submit</button
            >
          </div>
        </div>
      </div>
    </div>
  </div>
</main>
