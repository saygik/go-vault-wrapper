const { useState, useMemo, useEffect } = React;

const App = () => {
    const searchParams = new URLSearchParams(document.location.search)
    const [sMessage, SetSMessage] = useState('')
    const [errMsg, setErrMsg] = useState('');

    const token = useMemo(() => {
        return searchParams.get('token')
    }, [searchParams])

    const isReadyMsg = useMemo(() => { return sMessage === '' ? false : true }, [sMessage])

    useEffect(() => {
        setErrMsg('')
        $.get({
            url: "/api/token?token=" + token,
            success: function (response) {

                SetSMessage(response.data)
            },
            error: function (error) {
                console.log(error.responseText);
                setErrMsg(error.responseJSON.message)
            }
        })
    }, [token])
    //    console.log('token', token)
    return (
        <div className="container" style={{ marginTop: "50px" }}>
            <div className="col-xs-10 col-xs-offset-1 jumbotron pt-4">
                <h4 className="text-muted">Ваше сообщение</h4>
                { isReadyMsg && <div>
                    <div className="input-group input-group-sm mb-0">
                        <textarea className="form-control" value={sMessage} aria-label="With textarea" rows="3"></textarea>
                    </div>

                    <p className="text-muted mt-2 ml-1" style={{ fontSize: "0.8rem" }}>Это сообщение больше не отобразится. Скопируйте его перед закрытием этой страницы, если это необходимо. </p>

                </div>}
                {errMsg &&
                        <div class="alert alert-danger mt-4" role="alert">
                            {errMsg}
                        </div>
                    }
            </div>

        </div >
    )
}

ReactDOM.render(<App />, document.getElementById('app'));

