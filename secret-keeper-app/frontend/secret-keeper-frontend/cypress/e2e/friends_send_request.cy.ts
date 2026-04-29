describe('friends_send_request', () => {
  before(() => {
    cy.task('clearDB');
  });

  beforeEach(() => {
    cy.login('Alice', 'Alice123');
    cy.visit('/friends');
  });

  it('sends a friend request from the Add Friend tab', () => {
    cy.contains('.tab', 'Add Friend').click();
    cy.get('.add-section .field-input').type('Bob');
    cy.get('.send-btn').click();
    cy.get('.feedback').should('be.visible');
  });
});
