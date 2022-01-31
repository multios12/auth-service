import { Button, Card, CardActions, CardContent, FormControlLabel, Switch, TextField } from "@mui/material";
import { Box } from "@mui/system";

export default function Setting() {
    return <Box m={3}>
        <Card title="change password">
            <CardContent>
                <TextField name="oldPassword" label="old password" fullWidth />
                <TextField name="password" label="new password" fullWidth />
                <Button>change</Button>
            </CardContent>
        </Card>
        <Card title="authentication">
            <CardContent>
                <FormControlLabel control={<Switch />} label="finger print" />
            </CardContent>
        </Card>
    </Box>
}