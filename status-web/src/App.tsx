import React from 'react'
import { Pane, Text, Heading, Button, GitRepoIcon, Link, Alert, Spinner } from 'evergreen-ui'
import { Container, Row, Col } from 'react-grid-system'
import useAxios from 'axios-hooks'

import './App.css';
import StatisticsBoard from './StatisticsBoard'
import InfoBoard from './InfoBoard'
import { WssosksStatus } from './Status'

function App() {
  const [{ data, loading, error }, ] = useAxios<WssosksStatus>(
    window.location.protocol + "//" + window.location.host + "/api/status"
  )

  let content = null
  if (loading) {
    content = (
      <Row>
        <Col>
          <Pane flex={1} justifyContent="center" alignItems="center" display="flex">
            <Spinner/>
          </Pane>
        </Col>
      </Row>
    )
  } else if (error || !data) {
    content = (
      <Row>
        <Col offset={{ md: 3 }} md={6}>
          <Alert
            intent="danger"
            title={error? error.message: 'Error while loading server status.'}
          />
        </Col>
      </Row>
    )
  } else {
    content = (
      <>
        <Pane background="tint2" marginBottom={16} padding={24} >
          <Heading textAlign={"center"} is="h3">Information</Heading>
          <InfoBoard data={data.info}/>
        </Pane>
        <Pane background="tint1" padding={24} marginBottom={16}>
          <Heading textAlign={"center"} is="h3">Statistics</Heading>
          <StatisticsBoard data={data.statistics}/>
        </Pane>
      </>
    )
  }

  return (
    <Container fluid style={{ padding: 0 }} className="App">
      <Pane display="flex" padding={16} background="tint2" borderRadius={3}>
        <Pane flex={1} alignItems="center" display="flex">
          <Heading size={600}>wssocks status</Heading>
        </Pane>
        <Pane>
          <Link className="github-link" href="https://github.com/genshen/wssocks" target="_blank" marginRight={12}>
            <Button marginRight={8} appearance="minimal">
              <GitRepoIcon color="base" marginRight={8} />
              <Text color="dark">Github</Text>
            </Button>
          </Link>
        </Pane>
      </Pane>
      <Pane marginTop={20}>
        {content}
      </Pane>
    </Container>
  );
}

export default App;
