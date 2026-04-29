// before: set up a real Alice→Bob friendship so Remove is always present.
// beforeEach: re-login as Alice so each test starts with a clean dialog state.
describe('friends_confirm_dialog', () => {
  before(() => {
    cy.task('clearDB');
    // Alice sends a friend request to Bob
    cy.login('Alice', 'Alice123');
    cy.visit('/friends');
    cy.contains('.tab', 'Add Friend').click();
    cy.get('.add-section .field-input').type('Bob');
    cy.get('.send-btn').click();
    cy.get('.feedback').should('be.visible');
    // Bob accepts
    cy.login('Bob', 'Bob12345');
    cy.visit('/friends');
    cy.contains('.tab', 'Requests').click();
    cy.get('.friend-row').should('be.visible');
    cy.contains('button', 'Accept').click();
  });

  beforeEach(() => {
    cy.login('Alice', 'Alice123');
    cy.visit('/friends');
  });

  it('shows a confirmation dialog when clicking Remove on a friend', () => {
    cy.get('.action-btn.danger').first().click();
    cy.get('.confirm-overlay').should('be.visible');
    cy.get('.confirm-msg').should('be.visible');
    cy.get('.confirm-cancel').should('be.visible');
    cy.get('.confirm-ok').should('be.visible');
  });

  it('dismisses the dialog when Cancel is clicked', () => {
    cy.get('.action-btn.danger').first().click();
    cy.get('.confirm-overlay').should('be.visible');
    cy.get('.confirm-cancel').click();
    cy.get('.confirm-overlay').should('not.exist');
  });

  it('dismisses the dialog when clicking the overlay backdrop', () => {
    cy.get('.action-btn.danger').first().click();
    cy.get('.confirm-overlay').should('be.visible');
    cy.get('.confirm-overlay').click({ force: true });
    cy.get('.confirm-overlay').should('not.exist');
  });
});
