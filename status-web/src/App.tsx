import React from 'react';
import { Pane, Text, Heading, Button, Popover, Position, GitRepoIcon, Link } from 'evergreen-ui'
import { Container } from 'react-grid-system'

import './App.css';
import StatisticsBoard from './StatisticsBoard'
import InfoBoard from './InfoBoard'

function App() {
  return (
    <Container fluid style={{padding: 0}} className="App">
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
        <Pane background="tint2" marginBottom={16} padding={24} >
          <Heading textAlign={"center"} is="h3">Information</Heading>
          <InfoBoard/>
        </Pane>
        <Pane background="tint1" padding={24} marginBottom={16}>
          <Heading textAlign={"center"} is="h3">Statistics</Heading>
          <StatisticsBoard/>
        </Pane>
      </Pane>
    </Container>
  );
}

export default App;
