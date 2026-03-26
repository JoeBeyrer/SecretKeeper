describe('friends_message_button', () => {
  it('adds Bob as a friend, accepts from Bob, then messages from Alice', () => {
    cy.login('Alice', 'Alice123');
    cy.visit('/friends');
    cy.contains('.tab', 'Add Friend').click();
    cy.get('.add-section .form-control').type('Bob');
    cy.get('.send-button').click();
    cy.get('.feedback').should('be.visible');

    cy.visit('/login');
    cy.get('#username').clear().type('Bob');
    cy.get('#password').clear().type('Bob12345');
    cy.get('.login-button').click();
    cy.url().should('include', '/messaging');

    cy.visit('/friends');
    cy.contains('.tab', 'Requests').click();
    cy.get('.friend-row', { timeout: 10000 }).first().within(() => {
      cy.get('.action-btn.accept').click();
    });

    cy.visit('/login');
    cy.get('#username').clear().type('Alice');
    cy.get('#password').clear().type('Alice123');
    cy.get('.login-button').click();
    cy.url().should('include', '/messaging');

    cy.visit('/friends');
    cy.get('.friend-row', { timeout: 10000 }).first().within(() => {
      cy.get('.action-btn.message').click();
    });

    cy.url().should('include', '/messaging');
  });
});
