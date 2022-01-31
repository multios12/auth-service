import { AppBar, Container, Toolbar, Typography } from "@mui/material";
import React from "react";

export default class MuAppBar extends React.Component {
  render() {
    return (
      <AppBar position="static">
        <Container maxWidth="xl">
          <Toolbar disableGutters>
            <Typography sx={{ mr: 2 }}  variant="h6" component="div">
              modern-utopia.net
            </Typography>
            <Typography fontSize="small">In the beginning was the Name</Typography>
          </Toolbar>
        </Container>
      </AppBar>
    )
  }
}