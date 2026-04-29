describe('messaging_message_search', () => {
  before(() => {
    cy.login('Alice', 'Alice123');
    cy.visit('/messaging?chatWith=Bob');

    cy.get('.modal').should('be.visible');
    cy.get('.modal-title').then(($title) => {
      const title = $title.text().trim();
      if (title === 'Choose Room Key') {
        cy.contains('button', 'Create Conversation').click();
        cy.contains('.modal-title', 'Your Room Key').should('be.visible');
        cy.contains('button', "I've saved it").click();
      } else if (title === 'Your Room Key') {
        cy.contains('button', "I've saved it").click();
      } else {
        cy.get('.modal input[type="text"], .modal input[type="password"]').first().type('testkey123');
        cy.contains('button', 'Join').click();
      }
    });

    cy.get('.modal').should('not.exist');

    // Send a known message to search for
    cy.get('.message-input-bar input[placeholder="Type your message here..."]').type('searchable content xyz');
    cy.get('.message-input-bar .rectangle-1-3').click();
    cy.contains('.message-bubble.sent', 'searchable content xyz', { timeout: 10000 }).should('be.visible');
  });

  it('opens the search bar when the search icon is clicked', () => {
    cy.get('.search-toggle-btn').click();
    cy.get('.msg-search-bar').should('be.visible');
    cy.get('.msg-search-input').should('be.visible');
  });

  it('filters messages by search query and highlights matches', () => {
    cy.get('.msg-search-input').clear().type('searchable');
    cy.get('.msg-search-count').should('contain', '1 result');
    cy.get('.msg-highlight').should('be.visible').and('contain', 'searchable');
  });

  it('shows zero results for a non-matching query', () => {
    cy.get('.msg-search-input').clear().type('zzznomatch999');
    cy.get('.msg-search-count').should('contain', '0 results');
    cy.get('.messages-area .message-row').should('not.exist');
  });

  it('closes the search bar and restores all messages', () => {
    cy.get('.search-toggle-btn').click();
    cy.get('.msg-search-bar').should('not.exist');
    cy.contains('.message-bubble.sent', 'searchable content xyz').should('be.visible');
  });
});
